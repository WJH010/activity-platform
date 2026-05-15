package pipeline

import (
	"context"
	"event-platform/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SyncController RAG同步管理控制器
type SyncController struct {
	syncSvc *SyncService
}

// NewSyncController 创建同步管理控制器
func NewSyncController(syncSvc *SyncService) *SyncController {
	return &SyncController{syncSvc: syncSvc}
}

// FullSync 触发全量同步
// POST /api/admin/rag/sync/full
func (ctr *SyncController) FullSync(c *gin.Context) {
	// 异步执行，避免请求超时
	go func() {
		if err := ctr.syncSvc.FullSync(context.Background()); err != nil {
			logrus.Errorf("全量同步失败: %v", err)
		}
	}()

	utils.Success(c, "全量同步已触发", nil)
}

// GetSyncStatus 查看同步状态
// GET /api/admin/rag/sync/status
func (ctr *SyncController) GetSyncStatus(c *gin.Context) {
	status := ctr.syncSvc.GetSyncStatus()
	utils.Success(c, "success", status)
}
