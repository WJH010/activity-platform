package controller

import (
	"activity-platform/internal/agent/dto"
	"activity-platform/internal/agent/engine"
	"activity-platform/internal/agent/service"
	"activity-platform/internal/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AgentChatController Agent对话控制器
type AgentChatController struct {
	chatSvc service.AgentChatService
}

// NewAgentChatController 创建Agent对话控制器
func NewAgentChatController(chatSvc service.AgentChatService) *AgentChatController {
	return &AgentChatController{chatSvc: chatSvc}
}

// ChatStream 流式对话（SSE）
// POST /api/agent/chat
func (ctr *AgentChatController) ChatStream(ctx *gin.Context) {
	var req dto.ChatRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取当前用户ID
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 获取AuthToken（用于内部API调用）
	authToken := ""
	if authHeader := ctx.GetHeader("Authorization"); authHeader != "" {
		authToken = authHeader
	}

	// 调用ChatService
	eventCh, sessionID, err := ctr.chatSvc.ChatStream(ctx.Request.Context(), userID, authToken, req)
	if err != nil {
		logrus.Errorf("ChatStream调用失败: %v", err)
		utils.HandlerFunc(ctx, utils.NewSystemError(fmt.Errorf("启动对话失败: %w", err)))
		return
	}

	// 设置SSE响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no") // 禁用Nginx缓冲

	// 先推送session_id
	ctr.writeSSE(ctx, "session_id", sessionID)

	// 从eventCh读取事件并推送
	for event := range eventCh {
		switch event.Type {
		case engine.EventContent:
			chunk := dto.StreamChunk{
				Type:    "content",
				Content: event.Content,
			}
			ctr.writeSSE(ctx, "message", chunk)

		case engine.EventToolCall:
			chunk := dto.StreamChunk{
				Type:     "tool_call",
				ToolName: event.ToolName,
				Status:   "executing",
			}
			ctr.writeSSE(ctx, "message", chunk)

		case engine.EventToolResult:
			chunk := dto.StreamChunk{
				Type:     "tool_result",
				ToolName: event.ToolName,
				Result:   event.Result,
			}
			ctr.writeSSE(ctx, "message", chunk)

		case engine.EventDone:
			chunk := dto.StreamChunk{
				Type:      "done",
				SessionID: sessionID,
			}
			ctr.writeSSE(ctx, "message", chunk)
			return // 对话结束

		case engine.EventError:
			chunk := dto.StreamChunk{
				Type:    "error",
				Content: event.Error,
			}
			ctr.writeSSE(ctx, "message", chunk)
			return
		}
	}
}

// GetSessionMessages 获取会话消息历史
// GET /api/agent/sessions/:id/messages
func (ctr *AgentChatController) GetSessionMessages(ctx *gin.Context) {
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

	messages, err := ctr.chatSvc.LoadSessionMessages(ctx.Request.Context(), sessionID, userID)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "获取消息成功", messages)
}

// writeSSE 写入SSE数据
func (ctr *AgentChatController) writeSSE(ctx *gin.Context, event string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		logrus.Warnf("序列化SSE数据失败: %v", err)
		return
	}

	fmt.Fprintf(ctx.Writer, "event: %s\ndata: %s\n\n", event, string(dataBytes))
	ctx.Writer.(http.Flusher).Flush()
}

// writeSSEError 写入SSE错误
func (ctr *AgentChatController) writeSSEError(ctx *gin.Context, errMsg string) {
	fmt.Fprintf(ctx.Writer, "event: error\ndata: %s\n\n", errMsg)
	ctx.Writer.(http.Flusher).Flush()
}

// Ensure io is used
var _ = io.EOF
