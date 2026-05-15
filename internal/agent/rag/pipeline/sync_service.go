package pipeline

import (
	"context"
	"event-platform/internal/agent/rag/embedding"
	"event-platform/internal/agent/rag/milvus"
	articlerepo "event-platform/internal/article/repository"
	eventrepo "event-platform/internal/event/repository"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SyncService 数据同步服务：MySQL → Milvus
type SyncService struct {
	articleRepo   articlerepo.ArticleRepository
	eventRepo     eventrepo.EventRepository
	milvusRepo    *milvus.Repository
	embeddingSvc  embedding.EmbeddingService
	textProcessor *TextProcessor
	syncStateRepo *SyncStateRepository

	mu          sync.Mutex
	isSyncing   bool
	syncSummary string
}

// NewSyncService 创建数据同步服务
func NewSyncService(
	articleRepo articlerepo.ArticleRepository,
	eventRepo eventrepo.EventRepository,
	milvusRepo *milvus.Repository,
	embeddingSvc embedding.EmbeddingService,
	textProcessor *TextProcessor,
	db *gorm.DB,
) *SyncService {
	return &SyncService{
		articleRepo:   articleRepo,
		eventRepo:     eventRepo,
		milvusRepo:    milvusRepo,
		embeddingSvc:  embeddingSvc,
		textProcessor: textProcessor,
		syncStateRepo: NewSyncStateRepository(db),
	}
}

// ========== 公开方法 ==========

// FullSync 全量同步
func (s *SyncService) FullSync(ctx context.Context) error {
	s.mu.Lock()
	if s.isSyncing {
		s.mu.Unlock()
		return fmt.Errorf("同步正在进行中，请稍后")
	}
	s.isSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	startTime := time.Now()
	//logrus.Infof("[RAG同步] 开始全量同步...")

	// 全量同步文章
	articleCount, err := s.fullSync(ctx, "文章", milvus.ArticleCollectionName, s.fetchAllArticles)
	if err != nil {
		s.updateSyncState(startTime, "article", "失败", fmt.Sprintf("全量同步失败: %v", err), false)
		s.updateSyncState(startTime, "event", "跳过", "文章同步失败，跳过活动同步", false)
		s.syncSummary = fmt.Sprintf("全量同步失败: %v", err)
		return fmt.Errorf("全量同步文章失败: %w", err)
	}

	eventCount, err := s.fullSync(ctx, "活动", milvus.EventCollectionName, s.fetchAllEvents)
	if err != nil {
		s.updateSyncState(startTime, "article", "成功", fmt.Sprintf("全量同步完成: 文章%d篇", articleCount), true)
		s.updateSyncState(startTime, "event", "失败", fmt.Sprintf("全量同步失败: %v", err), false)
		s.syncSummary = fmt.Sprintf("全量同步部分失败: 文章%d篇成功, 活动失败: %v", articleCount, err)
		return fmt.Errorf("全量同步活动失败: %w", err)
	}

	summary := fmt.Sprintf("全量同步完成: 文章%d篇, 活动%d个, 耗时%v", articleCount, eventCount, time.Since(startTime))
	s.syncSummary = summary
	//logrus.Infof("[RAG同步] %s", summary)

	s.updateSyncState(startTime, "article", "成功", summary, true)
	s.updateSyncState(startTime, "event", "成功", summary, true)
	return nil
}

// IncrementalSync 增量同步
func (s *SyncService) IncrementalSync(ctx context.Context) error {
	s.mu.Lock()
	if s.isSyncing {
		s.mu.Unlock()
		return nil
	}
	s.isSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	startTime := time.Now()
	//logrus.Infof("[RAG同步] 开始增量同步...")

	articleSince := s.getLastSyncAtForName(ctx, "article")
	eventSince := s.getLastSyncAtForName(ctx, "event")
	//logrus.Infof("[RAG同步] 增量同步起始时间: 文章=%s, 活动=%s",
	//articleSince.Format("2006-01-02 15:04:05"), eventSince.Format("2006-01-02 15:04:05"))

	articleCount, err := s.incrementalSync(ctx, "文章", milvus.ArticleCollectionName, "article_id", articleSince, s.fetchUpdatedArticles)
	if err != nil {
		summary := fmt.Sprintf("增量同步失败: %v", err)
		s.syncSummary = summary
		s.updateSyncState(startTime, "article", "失败", summary, false)
		return err
	}
	s.updateSyncState(startTime, "article", "成功", fmt.Sprintf("增量同步: 文章%d篇", articleCount), true)

	eventCount, err := s.incrementalSync(ctx, "活动", milvus.EventCollectionName, "event_id", eventSince, s.fetchUpdatedEvents)
	if err != nil {
		summary := fmt.Sprintf("增量同步部分失败: 文章%d篇成功, 活动失败: %v", articleCount, err)
		s.syncSummary = summary
		s.updateSyncState(startTime, "event", "失败", err.Error(), false)
		return err
	}

	summary := fmt.Sprintf("增量同步完成: 文章%d篇, 活动%d个, 耗时%v", articleCount, eventCount, time.Since(startTime))
	s.syncSummary = summary
	//logrus.Infof("[RAG同步] %s", summary)

	s.updateSyncState(startTime, "event", "成功", summary, true)
	return nil
}

// StartIncrementalSync 启动定时增量同步
func (s *SyncService) StartIncrementalSync(ctx context.Context, intervalSeconds int) {
	if intervalSeconds <= 0 {
		intervalSeconds = 300
	}

	// 检查是否需要首次全量同步
	articleState, _ := s.syncStateRepo.Get(ctx, "article")
	eventState, _ := s.syncStateRepo.Get(ctx, "event")

	if articleState == nil || eventState == nil {
		//logrus.Infof("[RAG同步] 首次启动，执行全量同步...")
		if err := s.FullSync(ctx); err != nil {
			logrus.Errorf("[RAG同步] 首次全量同步失败: %v", err)
		}
	}
	// else {
	// 	logrus.Infof("[RAG同步] 已有同步记录，上次同步时间: 文章=%s, 活动=%s",
	// 		articleState.LastSyncAt.Format("2006-01-02 15:04:05"),
	// 		eventState.LastSyncAt.Format("2006-01-02 15:04:05"))
	// }

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	//logrus.Infof("[RAG同步] 启动定时增量同步，间隔 %d 秒", intervalSeconds)

	for {
		select {
		case <-ctx.Done():
			//logrus.Info("[RAG同步] 停止定时增量同步")
			return
		case <-ticker.C:
			if err := s.IncrementalSync(ctx); err != nil {
				logrus.Errorf("[RAG同步] 增量同步失败: %v", err)
			}
		}
	}
}

// GetSyncStatus 获取同步状态
func (s *SyncService) GetSyncStatus() map[string]interface{} {
	ctx := context.Background()
	articleState, _ := s.syncStateRepo.Get(ctx, "article")
	eventState, _ := s.syncStateRepo.Get(ctx, "event")

	s.mu.Lock()
	isSyncing := s.isSyncing
	summary := s.syncSummary
	s.mu.Unlock()

	status := map[string]interface{}{
		"is_syncing": isSyncing,
		"summary":    summary,
	}

	if articleState != nil {
		status["article_last_sync_at"] = articleState.LastSyncAt.Format("2006-01-02 15:04:05")
		status["article_status"] = articleState.Status
	}
	if eventState != nil {
		status["event_last_sync_at"] = eventState.LastSyncAt.Format("2006-01-02 15:04:05")
		status["event_status"] = eventState.Status
	}

	return status
}

// ========== 内部类型和方法 ==========

// syncItem 统一的同步数据项（数据源无关）
type syncItem struct {
	id        int64              // 数据ID
	isDeleted bool               // 是否已删除
	chunks    []milvus.ChunkData // 已填充元数据的分块数据
}

// fullSync 通用全量同步
func (s *SyncService) fullSync(
	ctx context.Context,
	sourceName string,
	collectionName string,
	fetchFn func(ctx context.Context, page, pageSize int) ([]syncItem, int64, error),
) (int, error) {
	batchSize := 50
	page := 1
	totalProcessed := 0

	for {
		items, total, err := fetchFn(ctx, page, batchSize)
		if err != nil {
			return totalProcessed, fmt.Errorf("查询%s列表失败 (page=%d): %w", sourceName, page, err)
		}
		if len(items) == 0 {
			break
		}

		// 收集所有分块和文本
		var allChunks []milvus.ChunkData
		var allTexts []string
		for _, item := range items {
			for _, chunk := range item.chunks {
				allTexts = append(allTexts, chunk.TextContent)
			}
			allChunks = append(allChunks, item.chunks...)
		}

		// 批量Embed
		if len(allTexts) > 0 {
			vectors, err := s.embeddingSvc.EmbedBatch(ctx, allTexts)
			if err != nil {
				return totalProcessed, fmt.Errorf("批量嵌入失败: %w", err)
			}
			if len(vectors) != len(allChunks) {
				return totalProcessed, fmt.Errorf("向量数量(%d)与分块数量(%d)不匹配", len(vectors), len(allChunks))
			}
			for i := range allChunks {
				allChunks[i].DenseVector = vectors[i]
			}
		}

		// 写入Milvus
		if err := s.milvusRepo.UpsertChunks(ctx, collectionName, allChunks); err != nil {
			return totalProcessed, fmt.Errorf("写入Milvus失败: %w", err)
		}

		totalProcessed += len(items)
		logrus.Infof("[RAG同步] %s全量同步进度: %d/%d", sourceName, totalProcessed, total)

		if int64(page*batchSize) >= total {
			break
		}
		page++
	}
	return totalProcessed, nil
}

// incrementalSync 通用增量同步
func (s *SyncService) incrementalSync(
	ctx context.Context,
	sourceName string,
	collectionName string,
	idField string,
	since time.Time,
	fetchFn func(ctx context.Context, since time.Time, pageSize, page int) ([]syncItem, int64, error),
) (int, error) {
	batchSize := 50
	page := 1
	totalProcessed := 0
	var deletedCount, updatedCount, vectorReusedCount int

	for {
		// 获取增量数据：新增或更新的文章数据
		items, total, err := fetchFn(ctx, since, batchSize, page)
		if err != nil {
			return totalProcessed, fmt.Errorf("增量查询%s失败 (page=%d): %w", sourceName, page, err)
		}
		if len(items) == 0 {
			break
		}

		// 逐块处理增量数据
		for _, item := range items {
			totalProcessed++

			// 已删除：从Milvus中删除
			if item.isDeleted {
				if err := s.milvusRepo.DeleteChunks(ctx, collectionName, idField, item.id); err != nil {
					logrus.Warnf("增量同步-删除%s分块失败 [id=%d]: %v", sourceName, item.id, err)
				} else {
					deletedCount++
				}
				continue
			}

			if len(item.chunks) == 0 {
				continue
			}

			// 查询Milvus中已有的分块数据
			existingChunks, err := s.milvusRepo.GetExistingChunks(ctx, collectionName, idField, item.id)
			if err != nil {
				logrus.Warnf("增量同步-查询已有%s分块失败 [id=%d]: %v", sourceName, item.id, err)
				existingChunks = nil
			}

			// 构建已有分块索引，用于内容对比
			existingMap := make(map[int32]milvus.ExistingChunkData)
			for _, ec := range existingChunks {
				existingMap[ec.ChunkIndex] = ec
			}

			// 对比内容，决定是否需要重新Embed
			var needEmbedTexts []string
			var needEmbedIndices []int
			for i, chunk := range item.chunks {
				if existing, ok := existingMap[chunk.ChunkIndex]; ok && existing.TextContent == chunk.TextContent {
					item.chunks[i].DenseVector = existing.DenseVector
					vectorReusedCount++
				} else {
					needEmbedTexts = append(needEmbedTexts, chunk.TextContent)
					needEmbedIndices = append(needEmbedIndices, i)
				}
			}

			// 对需要重新Embed的文本调用Embedding API
			if len(needEmbedTexts) > 0 {
				vectors, err := s.embeddingSvc.EmbedBatch(ctx, needEmbedTexts)
				if err != nil {
					logrus.Warnf("增量同步-%s嵌入失败 [id=%d]: %v", sourceName, item.id, err)
					continue
				}
				if len(vectors) != len(needEmbedIndices) {
					logrus.Warnf("增量同步-%s向量数量不匹配 [id=%d]: 期望%d, 实际%d", sourceName, item.id, len(needEmbedIndices), len(vectors))
					continue
				}
				for i, idx := range needEmbedIndices {
					item.chunks[idx].DenseVector = vectors[i]
				}
			}

			// 先删除旧数据（处理chunk数量减少的情况），再Upsert新数据
			if len(existingChunks) > 0 {
				_ = s.milvusRepo.DeleteChunks(ctx, collectionName, idField, item.id)
			}
			if err := s.milvusRepo.UpsertChunks(ctx, collectionName, item.chunks); err != nil {
				logrus.Warnf("增量同步-Upsert%s分块失败 [id=%d]: %v", sourceName, item.id, err)
				continue
			}

			updatedCount++
		}

		logrus.Infof("[RAG同步] %s增量同步进度: %d/%d (更新%d, 删除%d, 向量复用%d)",
			sourceName, totalProcessed, total, updatedCount, deletedCount, vectorReusedCount)

		if int64(page*batchSize) >= total {
			break
		}
		page++
	}

	logrus.Infof("[RAG同步] %s增量同步完成: 总计%d条 (更新%d, 删除%d, 向量复用%d)",
		sourceName, totalProcessed, updatedCount, deletedCount, vectorReusedCount)
	return totalProcessed, nil
}

// ========== 数据获取函数 ==========

// fetchAllArticles 全量获取文章数据
func (s *SyncService) fetchAllArticles(ctx context.Context, page, pageSize int) ([]syncItem, int64, error) {
	articles, total, err := s.articleRepo.List(ctx, page, pageSize, "", "", "", "", 0, "")
	if err != nil {
		return nil, 0, err
	}

	items := make([]syncItem, 0, len(articles))
	for _, art := range articles {
		content, err := s.articleRepo.GetArticleContent(ctx, art.ArticleID)
		if err != nil {
			logrus.Warnf("获取文章详情失败 [id=%d]: %v", art.ArticleID, err)
			continue
		}

		chunks := s.textProcessor.ProcessDocument(art.ArticleTitle, content.ArticleContent, art.BriefContent)
		chunkData := make([]milvus.ChunkData, 0, len(chunks))
		for _, chunk := range chunks {
			chunkData = append(chunkData, milvus.ChunkData{
				ID:           int64(art.ArticleID),
				ChunkIndex:   int32(chunk.Index),
				TextContent:  chunk.Content,
				ArticleTitle: art.ArticleTitle,
				BriefContent: truncate(art.BriefContent, 1000),
				FieldType:    content.FieldType,
				ArticleType:  content.ArticleTypeCode,
				IsSelection:  int32(art.IsSelection),
				CreateTime:   art.ReleaseTime.Unix(),
			})
		}
		items = append(items, syncItem{id: int64(art.ArticleID), chunks: chunkData})
	}
	return items, total, nil
}

// fetchAllEvents 全量获取活动数据
func (s *SyncService) fetchAllEvents(ctx context.Context, page, pageSize int) ([]syncItem, int64, error) {
	events, total, err := s.eventRepo.List(ctx, page, pageSize, "", "", "")
	if err != nil {
		return nil, 0, err
	}

	items := make([]syncItem, 0, len(events))
	for _, evt := range events {
		detail, err := s.eventRepo.GetEventDetail(ctx, evt.ID)
		if err != nil {
			logrus.Warnf("获取活动详情失败 [id=%d]: %v", evt.ID, err)
			continue
		}

		chunks := s.textProcessor.ProcessDocument(evt.Title, detail.Detail, "")
		chunkData := make([]milvus.ChunkData, 0, len(chunks))
		for _, chunk := range chunks {
			chunkData = append(chunkData, milvus.ChunkData{
				ID:             int64(evt.ID),
				ChunkIndex:     int32(chunk.Index),
				TextContent:    chunk.Content,
				EventTitle:     evt.Title,
				EventStatus:    computeEventStatus(evt.RegistrationStartTime, evt.RegistrationEndTime),
				EventAddress:   evt.EventAddress,
				EventStartTime: evt.EventStartTime.Unix(),
				EventEndTime:   evt.EventEndTime.Unix(),
			})
		}
		items = append(items, syncItem{id: int64(evt.ID), chunks: chunkData})
	}
	return items, total, nil
}

// fetchUpdatedArticles 增量获取文章变更数据
func (s *SyncService) fetchUpdatedArticles(ctx context.Context, since time.Time, pageSize, page int) ([]syncItem, int64, error) {
	// 查询指定时间后新增或更新的文章数据
	articles, total, err := s.articleRepo.ListUpdatedSince(ctx, since, pageSize, page)
	if err != nil {
		return nil, 0, err
	}

	items := make([]syncItem, 0, len(articles))
	// 处理文章数据：清洗文章内容，将文章内容转换为分块数据
	for _, art := range articles {
		// 清洗文章内容
		chunks := s.textProcessor.ProcessDocument(art.ArticleTitle, art.ArticleContent, art.BriefContent)
		chunkData := make([]milvus.ChunkData, 0, len(chunks))
		for _, chunk := range chunks {
			chunkData = append(chunkData, milvus.ChunkData{
				ID:           int64(art.ArticleID),
				ChunkIndex:   int32(chunk.Index),
				TextContent:  chunk.Content,
				ArticleTitle: art.ArticleTitle,
				BriefContent: truncate(art.BriefContent, 1000),
				FieldType:    art.FieldType,
				ArticleType:  art.ArticleType,
				IsSelection:  int32(art.IsSelection),
				CreateTime:   art.ReleaseTime.Unix(),
			})
		}
		items = append(items, syncItem{
			id:        int64(art.ArticleID),
			isDeleted: strings.ToUpper(art.IsDeleted) == "Y",
			chunks:    chunkData,
		})
	}
	return items, total, nil
}

// fetchUpdatedEvents 增量获取活动变更数据
func (s *SyncService) fetchUpdatedEvents(ctx context.Context, since time.Time, pageSize, page int) ([]syncItem, int64, error) {
	events, total, err := s.eventRepo.ListUpdatedSince(ctx, since, pageSize, page)
	if err != nil {
		return nil, 0, err
	}

	items := make([]syncItem, 0, len(events))
	for _, evt := range events {
		chunks := s.textProcessor.ProcessDocument(evt.Title, evt.Detail, "")
		chunkData := make([]milvus.ChunkData, 0, len(chunks))
		for _, chunk := range chunks {
			chunkData = append(chunkData, milvus.ChunkData{
				ID:             int64(evt.ID),
				ChunkIndex:     int32(chunk.Index),
				TextContent:    chunk.Content,
				EventTitle:     evt.Title,
				EventStatus:    computeEventStatus(evt.RegistrationStartTime, evt.RegistrationEndTime),
				EventAddress:   evt.EventAddress,
				EventStartTime: evt.EventStartTime.Unix(),
				EventEndTime:   evt.EventEndTime.Unix(),
			})
		}
		items = append(items, syncItem{
			id:        int64(evt.ID),
			isDeleted: strings.ToUpper(evt.IsDeleted) == "Y",
			chunks:    chunkData,
		})
	}
	return items, total, nil
}

// ========== 工具方法 ==========

// getLastSyncAtForName 获取指定数据源的上次同步时间
func (s *SyncService) getLastSyncAtForName(ctx context.Context, name string) time.Time {
	state, _ := s.syncStateRepo.Get(ctx, name)
	if state != nil {
		return state.LastSyncAt
	}
	return time.Time{}
}

// updateSyncState 更新同步状态到数据库
func (s *SyncService) updateSyncState(LastSyncAt time.Time, name string, status string, logMsg string, updateLastSyncAt bool) {
	ctx := context.Background()
	state := &SyncState{
		LastSyncAt: LastSyncAt,
		Name:       name,
		Status:     status,
		Log:        logMsg,
	}
	if updateLastSyncAt {
		// state.LastSyncAt = time.Now()
		if err := s.syncStateRepo.Upsert(ctx, state); err != nil {
			logrus.Warnf("[RAG同步] 更新同步状态失败 [%s]: %v", name, err)
		}
	} else {
		if err := s.syncStateRepo.UpdateStatusOnly(ctx, state); err != nil {
			logrus.Warnf("[RAG同步] 更新同步状态失败 [%s]: %v", name, err)
		}
	}
}

// computeEventStatus 根据报名时间计算活动状态
func computeEventStatus(registrationStart, registrationEnd time.Time) string {
	now := time.Now()
	if now.Before(registrationStart) {
		return "NotBegun"
	}
	if now.After(registrationEnd) {
		return "Completed"
	}
	return "InProgress"
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}
