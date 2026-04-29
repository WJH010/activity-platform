package controller

import (
	"activity-platform/internal/agent/service"
	"activity-platform/internal/utils"

	"github.com/gin-gonic/gin"
)

// AgentSessionController 会话管理控制器
type AgentSessionController struct {
	sessionSvc service.SessionService
}

// NewAgentSessionController 创建会话管理控制器
func NewAgentSessionController(sessionSvc service.SessionService) *AgentSessionController {
	return &AgentSessionController{sessionSvc: sessionSvc}
}

// ListSessions 获取当前用户的会话列表
// GET /api/agent/sessions
func (ctr *AgentSessionController) ListSessions(ctx *gin.Context) {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	sessions, err := ctr.sessionSvc.ListByUserID(ctx.Request.Context(), userID)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "获取会话列表成功", sessions)
}

// DeleteSession 软删除会话
// DELETE /api/agent/sessions/:id
func (ctr *AgentSessionController) DeleteSession(ctx *gin.Context) {
	sessionID := ctx.Param("id")
	if sessionID == "" {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "缺少会话ID"))
		return
	}

	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	if err := ctr.sessionSvc.SoftDeleteSession(ctx.Request.Context(), sessionID, userID); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "删除会话成功", nil)
}
