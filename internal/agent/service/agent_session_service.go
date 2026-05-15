package service

import (
	"context"
	"encoding/json"
	"event-platform/internal/agent/dto"
	"event-platform/internal/agent/model"
	"event-platform/internal/agent/repository"
	"event-platform/internal/utils"

	"github.com/google/uuid"
)

// SessionService 会话管理服务接口
type SessionService interface {
	// GetOrCreateSession 获取或创建会话
	GetOrCreateSession(ctx context.Context, userID int, sessionID string, llmConfigID *int) (*model.Session, error)
	// ListByUserID 获取用户会话列表
	ListByUserID(ctx context.Context, userID int) ([]dto.SessionResponse, error)
	// GetSessionMessages 获取会话消息历史
	GetSessionMessages(ctx context.Context, sessionID string, userID int) ([]dto.MessageResponse, error)
	// SoftDeleteSession 软删除会话
	SoftDeleteSession(ctx context.Context, sessionID string, userID int) error
	// UpdateSessionTitle 更新会话标题
	UpdateSessionTitle(ctx context.Context, sessionID string, title string) error
	// SaveMessage 保存消息
	SaveMessage(ctx context.Context, message *model.Message) error
	// SaveMessages 批量保存消息
	SaveMessages(ctx context.Context, messages []model.Message) error
}

type sessionServiceImpl struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
}

func NewSessionService(
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
) SessionService {
	return &sessionServiceImpl{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

func (svc *sessionServiceImpl) GetOrCreateSession(ctx context.Context, userID int, sessionID string, llmConfigID *int) (*model.Session, error) {
	// 如果提供了sessionID，尝试获取已有会话
	if sessionID != "" {
		session, err := svc.sessionRepo.GetByID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if session != nil && session.UserID == userID {
			return session, nil
		}
	}

	// 创建新会话
	session := &model.Session{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       "",
		LLMConfigID: llmConfigID,
		IsDeleted:   utils.DeletedFlagNo,
	}
	if err := svc.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (svc *sessionServiceImpl) ListByUserID(ctx context.Context, userID int) ([]dto.SessionResponse, error) {
	sessions, err := svc.sessionRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, dto.SessionResponse{
			ID:    s.ID,
			Title: s.Title,
		})
	}
	return result, nil
}

func (svc *sessionServiceImpl) GetSessionMessages(ctx context.Context, sessionID string, userID int) ([]dto.MessageResponse, error) {
	// 验证会话属于该用户
	session, err := svc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "会话不存在")
	}

	// 排除 role=tool 的消息（前端展示不需要）
	messages, err := svc.messageRepo.ListBySessionIDExcludeTool(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.MessageResponse, 0, len(messages))
	for _, m := range messages {
		msgResp := dto.MessageResponse{
			ID:        m.ID,
			Role:      m.Role,
			Content:   m.Content,
			SkillName: m.SkillName,
		}

		// 解析assistant消息中的tool_calls JSON
		if m.Role == "assistant" && m.ToolCalls != "" {
			var toolCalls interface{}
			if err := json.Unmarshal([]byte(m.ToolCalls), &toolCalls); err == nil {
				msgResp.ToolCalls = toolCalls
			}
		}

		// 填充skill_result
		if m.SkillResult != "" {
			var skillResult interface{}
			if err := json.Unmarshal([]byte(m.SkillResult), &skillResult); err == nil {
				msgResp.SkillResult = skillResult
			}
		}

		result = append(result, msgResp)
	}
	return result, nil
}

func (svc *sessionServiceImpl) SoftDeleteSession(ctx context.Context, sessionID string, userID int) error {
	session, err := svc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil || session.UserID != userID {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "会话不存在")
	}
	return svc.sessionRepo.SoftDelete(ctx, sessionID)
}

func (svc *sessionServiceImpl) UpdateSessionTitle(ctx context.Context, sessionID string, title string) error {
	return svc.sessionRepo.Update(ctx, sessionID, map[string]interface{}{
		"title": title,
	})
}

func (svc *sessionServiceImpl) SaveMessage(ctx context.Context, message *model.Message) error {
	return svc.messageRepo.Create(ctx, message)
}

func (svc *sessionServiceImpl) SaveMessages(ctx context.Context, messages []model.Message) error {
	return svc.messageRepo.BatchCreate(ctx, messages)
}
