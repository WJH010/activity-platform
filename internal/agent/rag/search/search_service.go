package search

import (
	"context"
	"fmt"
	"strings"

	"activity-platform/internal/agent/rag/embedding"
	"activity-platform/internal/agent/rag/milvus"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/sirupsen/logrus"
)

// SearchType 检索类型
type SearchType string

const (
	SearchTypeArticle SearchType = "article" // 文章检索
	SearchTypeEvent   SearchType = "event"   // 活动检索
	SearchTypeAll     SearchType = "all"     // 同时检索文章和活动
)

// SearchResult 单条检索结果
type SearchResult struct {
	// 通用字段
	ID          int64   `json:"id"`           // 文章ID或活动ID
	ChunkIndex  int32   `json:"chunk_index"`  // 分块序号
	TextContent string  `json:"text_content"` // 匹配的文本片段
	Score       float32 `json:"score"`        // 相关性得分

	// 文章字段
	ArticleTitle string `json:"article_title,omitempty"`
	BriefContent string `json:"brief_content,omitempty"`
	FieldType    string `json:"field_type,omitempty"`
	ArticleType  string `json:"article_type,omitempty"`
	IsSelection  int32  `json:"is_selection,omitempty"`
	CreateTime   int64  `json:"create_time,omitempty"`

	// 活动字段
	EventTitle     string `json:"event_title,omitempty"`
	EventStatus    string `json:"event_status,omitempty"`
	EventAddress   string `json:"event_address,omitempty"`
	EventStartTime int64  `json:"event_start_time,omitempty"`
	EventEndTime   int64  `json:"event_end_time,omitempty"`

	// 元数据
	SourceType SearchType `json:"source_type"` // 结果来源：article 或 event
}

// SearchResponse 检索响应
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"` // 结果数量
	Query   string         `json:"query"` // 原始查询
}

// SearchService 混合检索服务
type SearchService struct {
	milvusRepo   *milvus.Repository
	embeddingSvc embedding.EmbeddingService
}

// NewSearchService 创建混合检索服务
func NewSearchService(
	milvusRepo *milvus.Repository,
	embeddingSvc embedding.EmbeddingService,
) *SearchService {
	return &SearchService{
		milvusRepo:   milvusRepo,
		embeddingSvc: embeddingSvc,
	}
}

// Search 执行混合检索（dense + BM25 + RRF）
func (s *SearchService) Search(ctx context.Context, query string, searchType SearchType, topK int) (*SearchResponse, error) {
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	// 1. 将查询文本向量化
	queryVector, err := s.embeddingSvc.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}

	var allResults []SearchResult

	// 2. 根据检索类型执行搜索
	if searchType == SearchTypeArticle || searchType == SearchTypeAll {
		articleResults, err := s.searchCollection(ctx, milvus.ArticleCollectionName, query, queryVector, topK, SearchTypeArticle)
		if err != nil {
			logrus.Warnf("文章混合检索失败: %v", err)
		} else {
			allResults = append(allResults, articleResults...)
		}
	}

	if searchType == SearchTypeEvent || searchType == SearchTypeAll {
		eventResults, err := s.searchCollection(ctx, milvus.EventCollectionName, query, queryVector, topK, SearchTypeEvent)
		if err != nil {
			logrus.Warnf("活动混合检索失败: %v", err)
		} else {
			allResults = append(allResults, eventResults...)
		}
	}

	return &SearchResponse{
		Results: allResults,
		Total:   len(allResults),
		Query:   query,
	}, nil
}

// searchCollection 对单个Collection执行混合检索
func (s *SearchService) searchCollection(
	ctx context.Context,
	collectionName string,
	queryText string,
	queryVector []float32,
	topK int,
	sourceType SearchType,
) ([]SearchResult, error) {
	// 每路召回的候选数量（比最终topK大，给RRF更多融合空间）
	candidateLimit := topK * 3

	// dense向量检索请求
	denseReq := milvusclient.NewAnnRequest("dense_vector", candidateLimit,
		entity.FloatVector(queryVector),
	).WithSearchParam("metric_type", "COSINE"). // 明确度量类型为余弦相似度
							WithSearchParam("radius", "0.3") // 设置宽松阈值

	// BM25全文检索请求（直接传原始文本，Milvus自动转为稀疏向量）
	bm25Req := milvusclient.NewAnnRequest("sparse_vector", candidateLimit,
		entity.Text(queryText),
	)

	// 输出字段
	var outputFields []string
	if sourceType == SearchTypeArticle {
		outputFields = []string{"article_id", "chunk_index", "text_content",
			"article_title", "brief_content", "field_type", "article_type",
			"is_selection", "create_time"}
	} else {
		outputFields = []string{"event_id", "chunk_index", "text_content",
			"event_title", "event_status", "event_address",
			"event_start_time", "event_end_time"}
	}

	// 混合检索：dense + BM25 + RRF
	resultSets, err := s.milvusRepo.GetClient().HybridSearch(ctx, milvusclient.NewHybridSearchOption(
		collectionName, topK, denseReq, bm25Req,
	).WithReranker(milvusclient.NewRRFReranker()).
		WithOutputFields(outputFields...))
	if err != nil {
		return nil, fmt.Errorf("混合检索失败 [%s]: %w", collectionName, err)
	}

	if len(resultSets) == 0 {
		return nil, nil
	}

	// 解析结果
	return s.parseResults(&resultSets[0], sourceType)
}

// parseResults 解析Milvus返回的ResultSet
func (s *SearchService) parseResults(rs *milvusclient.ResultSet, sourceType SearchType) ([]SearchResult, error) {
	resultCount := rs.Len()
	if resultCount == 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, resultCount)

	for i := 0; i < resultCount; i++ {
		sr := SearchResult{SourceType: sourceType}

		if sourceType == SearchTypeArticle {
			sr.ID = mustGetInt64(rs, "article_id", i)
			sr.ArticleTitle = mustGetString(rs, "article_title", i)
			sr.BriefContent = mustGetString(rs, "brief_content", i)
			sr.FieldType = mustGetString(rs, "field_type", i)
			sr.ArticleType = mustGetString(rs, "article_type", i)
			sr.IsSelection = mustGetInt32(rs, "is_selection", i)
			sr.CreateTime = mustGetInt64(rs, "create_time", i)
		} else {
			sr.ID = mustGetInt64(rs, "event_id", i)
			sr.EventTitle = mustGetString(rs, "event_title", i)
			sr.EventStatus = mustGetString(rs, "event_status", i)
			sr.EventAddress = mustGetString(rs, "event_address", i)
			sr.EventStartTime = mustGetInt64(rs, "event_start_time", i)
			sr.EventEndTime = mustGetInt64(rs, "event_end_time", i)
		}

		sr.ChunkIndex = mustGetInt32(rs, "chunk_index", i)
		sr.TextContent = mustGetString(rs, "text_content", i)

		results = append(results, sr)
	}

	return results, nil
}

// mustGetInt64 安全地从ResultSet中获取int64字段值
func mustGetInt64(rs *milvusclient.ResultSet, fieldName string, idx int) int64 {
	col := rs.GetColumn(fieldName)
	if col == nil {
		return 0
	}
	if c, ok := col.(*column.ColumnInt64); ok {
		if val, err := c.Value(idx); err == nil {
			return val
		}
	}
	return 0
}

// mustGetInt32 安全地从ResultSet中获取int32字段值
func mustGetInt32(rs *milvusclient.ResultSet, fieldName string, idx int) int32 {
	col := rs.GetColumn(fieldName)
	if col == nil {
		return 0
	}
	if c, ok := col.(*column.ColumnInt32); ok {
		if val, err := c.Value(idx); err == nil {
			return val
		}
	}
	return 0
}

// mustGetString 安全地从ResultSet中获取string字段值
func mustGetString(rs *milvusclient.ResultSet, fieldName string, idx int) string {
	col := rs.GetColumn(fieldName)
	if col == nil {
		return ""
	}
	if c, ok := col.(*column.ColumnVarChar); ok {
		if val, err := c.Value(idx); err == nil {
			return val
		}
	}
	return ""
}

// DeduplicateAndAggregate 对结果按ID去重并聚合（同一文档的多个chunk合并为一条摘要）
// 返回的每条结果包含文本片段列表，方便LLM理解
func DeduplicateAndAggregate(results []SearchResult) []AggregatedResult {
	idMap := make(map[string]*AggregatedResult)

	for _, r := range results {
		key := fmt.Sprintf("%s_%d", r.SourceType, r.ID)
		if existing, ok := idMap[key]; ok {
			// 同一文档的多个chunk，追加文本片段
			existing.Chunks = append(existing.Chunks, r.TextContent)
		} else {
			ar := AggregatedResult{
				ID:             r.ID,
				SourceType:     r.SourceType,
				ArticleTitle:   r.ArticleTitle,
				BriefContent:   r.BriefContent,
				FieldType:      r.FieldType,
				ArticleType:    r.ArticleType,
				IsSelection:    r.IsSelection,
				CreateTime:     r.CreateTime,
				EventTitle:     r.EventTitle,
				EventStatus:    r.EventStatus,
				EventAddress:   r.EventAddress,
				EventStartTime: r.EventStartTime,
				EventEndTime:   r.EventEndTime,
				Chunks:         []string{r.TextContent},
			}
			idMap[key] = &ar
		}
	}

	aggregated := make([]AggregatedResult, 0, len(idMap))
	for _, ar := range idMap {
		ar.TextSummary = strings.Join(ar.Chunks, "\n---\n")
		aggregated = append(aggregated, *ar)
	}
	return aggregated
}

// AggregatedResult 聚合后的检索结果（同一文档的多chunk合并）
type AggregatedResult struct {
	ID          int64      `json:"id"`
	SourceType  SearchType `json:"source_type"`
	Chunks      []string   `json:"chunks"`       // 所有匹配的文本片段
	TextSummary string     `json:"text_summary"` // 合并后的文本摘要

	// 文章字段
	ArticleTitle string `json:"article_title,omitempty"`
	BriefContent string `json:"brief_content,omitempty"`
	FieldType    string `json:"field_type,omitempty"`
	ArticleType  string `json:"article_type,omitempty"`
	IsSelection  int32  `json:"is_selection,omitempty"`
	CreateTime   int64  `json:"create_time,omitempty"`

	// 活动字段
	EventTitle     string `json:"event_title,omitempty"`
	EventStatus    string `json:"event_status,omitempty"`
	EventAddress   string `json:"event_address,omitempty"`
	EventStartTime int64  `json:"event_start_time,omitempty"`
	EventEndTime   int64  `json:"event_end_time,omitempty"`
}
