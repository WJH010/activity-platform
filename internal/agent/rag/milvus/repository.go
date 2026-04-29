package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/sirupsen/logrus"
)

const maxBatchSize = 500

// ChunkData 统一分块数据模型（用于写入Milvus）
// 各数据源填充通用字段 + 自身对应的扩展字段，未使用字段保持零值
type ChunkData struct {
	// 通用字段
	ID          int64
	ChunkIndex  int32
	TextContent string
	DenseVector []float32

	// 文章扩展字段
	ArticleTitle string
	BriefContent string
	FieldType    string
	ArticleType  string
	IsSelection  int32
	CreateTime   int64

	// 活动扩展字段
	EventTitle     string
	EventStatus    string
	EventAddress   string
	EventStartTime int64
	EventEndTime   int64
}

// ExistingChunkData Milvus中已有的分块数据（用于增量对比）
type ExistingChunkData struct {
	ChunkIndex  int32
	TextContent string
	DenseVector []float32
}

// Repository Milvus数据读写接口
type Repository struct {
	client *milvusclient.Client
}

// NewRepository 创建Milvus数据仓库
func NewRepository(client *milvusclient.Client) *Repository {
	return &Repository{client: client}
}

// GetExistingChunks 获取指定ID的已有分块（通用方法）
func (r *Repository) GetExistingChunks(ctx context.Context, collectionName, idField string, id int64) ([]ExistingChunkData, error) {
	resultSet, err := r.client.Query(ctx, milvusclient.NewQueryOption(collectionName).
		WithFilter(fmt.Sprintf("%s == %d", idField, id)).
		WithOutputFields("chunk_index", "text_content", "dense_vector"))
	if err != nil {
		return nil, fmt.Errorf("查询已有分块失败 [%s %s=%d]: %w", collectionName, idField, id, err)
	}

	count := resultSet.Len()
	if count == 0 {
		return nil, nil
	}

	chunks := make([]ExistingChunkData, 0, count)
	for i := 0; i < count; i++ {
		chunk := ExistingChunkData{}
		if col, ok := resultSet.GetColumn("chunk_index").(*column.ColumnInt32); ok {
			if v, err := col.Value(i); err == nil {
				chunk.ChunkIndex = v
			}
		}
		if col, ok := resultSet.GetColumn("text_content").(*column.ColumnVarChar); ok {
			if v, err := col.Value(i); err == nil {
				chunk.TextContent = v
			}
		}
		if col, ok := resultSet.GetColumn("dense_vector").(*column.ColumnFloatVector); ok {
			if v, err := col.Value(i); err == nil {
				chunk.DenseVector = v
			}
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// DeleteChunks 删除指定ID的所有分块（通用方法）
func (r *Repository) DeleteChunks(ctx context.Context, collectionName, idField string, id int64) error {
	_, err := r.client.Delete(ctx, milvusclient.NewDeleteOption(collectionName).
		WithExpr(fmt.Sprintf("%s == %d", idField, id)),
	)
	if err != nil {
		return fmt.Errorf("从Milvus删除分块失败 [%s %s=%d]: %w", collectionName, idField, id, err)
	}
	logrus.Debugf("成功从Milvus删除分块 [%s %s=%d]", collectionName, idField, id)
	return nil
}

// UpsertChunks 批量写入/更新分块（通用方法，根据collectionName构建不同列结构）
func (r *Repository) UpsertChunks(ctx context.Context, collectionName string, chunks []ChunkData) error {
	if len(chunks) == 0 {
		return nil
	}

	for batchStart := 0; batchStart < len(chunks); batchStart += maxBatchSize {
		batchEnd := batchStart + maxBatchSize
		if batchEnd > len(chunks) {
			batchEnd = len(chunks)
		}
		batch := chunks[batchStart:batchEnd]

		// 通用列
		pks := make([]string, 0, len(batch))
		ids := make([]int64, 0, len(batch))
		chunkIndexes := make([]int32, 0, len(batch))
		textContents := make([]string, 0, len(batch))
		denseVectors := make([][]float32, 0, len(batch))

		for _, c := range batch {
			pks = append(pks, fmt.Sprintf("%d_%d", c.ID, c.ChunkIndex))
			ids = append(ids, c.ID)
			chunkIndexes = append(chunkIndexes, c.ChunkIndex)
			textContents = append(textContents, c.TextContent)
			denseVectors = append(denseVectors, c.DenseVector)
		}

		var columns []column.Column
		switch collectionName {
		case ArticleCollectionName:
			articleTitles := make([]string, 0, len(batch))
			briefContents := make([]string, 0, len(batch))
			fieldTypes := make([]string, 0, len(batch))
			articleTypes := make([]string, 0, len(batch))
			isSelections := make([]int32, 0, len(batch))
			createTimes := make([]int64, 0, len(batch))
			for _, c := range batch {
				articleTitles = append(articleTitles, c.ArticleTitle)
				briefContents = append(briefContents, c.BriefContent)
				fieldTypes = append(fieldTypes, c.FieldType)
				articleTypes = append(articleTypes, c.ArticleType)
				isSelections = append(isSelections, c.IsSelection)
				createTimes = append(createTimes, c.CreateTime)
			}
			columns = []column.Column{
				column.NewColumnVarChar("pk", pks),
				column.NewColumnInt64("article_id", ids),
				column.NewColumnInt32("chunk_index", chunkIndexes),
				column.NewColumnVarChar("text_content", textContents),
				column.NewColumnFloatVector("dense_vector", VectorDimension, denseVectors),
				column.NewColumnVarChar("article_title", articleTitles),
				column.NewColumnVarChar("brief_content", briefContents),
				column.NewColumnVarChar("field_type", fieldTypes),
				column.NewColumnVarChar("article_type", articleTypes),
				column.NewColumnInt32("is_selection", isSelections),
				column.NewColumnInt64("create_time", createTimes),
			}

		case EventCollectionName:
			eventTitles := make([]string, 0, len(batch))
			eventStatuses := make([]string, 0, len(batch))
			eventAddresses := make([]string, 0, len(batch))
			eventStartTimes := make([]int64, 0, len(batch))
			eventEndTimes := make([]int64, 0, len(batch))
			for _, c := range batch {
				eventTitles = append(eventTitles, c.EventTitle)
				eventStatuses = append(eventStatuses, c.EventStatus)
				eventAddresses = append(eventAddresses, c.EventAddress)
				eventStartTimes = append(eventStartTimes, c.EventStartTime)
				eventEndTimes = append(eventEndTimes, c.EventEndTime)
			}
			columns = []column.Column{
				column.NewColumnVarChar("pk", pks),
				column.NewColumnInt64("event_id", ids),
				column.NewColumnInt32("chunk_index", chunkIndexes),
				column.NewColumnVarChar("text_content", textContents),
				column.NewColumnFloatVector("dense_vector", VectorDimension, denseVectors),
				column.NewColumnVarChar("event_title", eventTitles),
				column.NewColumnVarChar("event_status", eventStatuses),
				column.NewColumnVarChar("event_address", eventAddresses),
				column.NewColumnInt64("event_start_time", eventStartTimes),
				column.NewColumnInt64("event_end_time", eventEndTimes),
			}

		default:
			return fmt.Errorf("未知的Collection: %s", collectionName)
		}

		_, err := r.client.Upsert(ctx, milvusclient.NewColumnBasedInsertOption(collectionName, columns...))
		if err != nil {
			return fmt.Errorf("Upsert分块到Milvus失败 [%s] (batch %d-%d): %w", collectionName, batchStart, batchEnd, err)
		}
		logrus.Debugf("成功Upsert %d 条分块到Milvus [%s] (batch %d-%d)", len(batch), collectionName, batchStart, batchEnd)
	}

	return nil
}

// GetClient 获取底层Milvus客户端（用于Search/HybridSearch等高级操作）
func (r *Repository) GetClient() *milvusclient.Client {
	return r.client
}
