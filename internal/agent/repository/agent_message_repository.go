package repository

import (
	"activity-platform/internal/agent/model"
	"activity-platform/internal/utils"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// MessageRepository 消息数据访问接口
type MessageRepository interface {
	// ListBySessionID 获取会话消息列表
	ListBySessionID(ctx context.Context, sessionID string) ([]model.Message, error)
	// Create 创建消息
	Create(ctx context.Context, message *model.Message) error
	// BatchCreate 批量创建消息
	BatchCreate(ctx context.Context, messages []model.Message) error
	// DeleteBySessionID 删除会话所有消息
	DeleteBySessionID(ctx context.Context, sessionID string) error
	// ListBySessionIDExcludeTool 获取会话消息列表（排除tool角色消息，用于前端展示）
	ListBySessionIDExcludeTool(ctx context.Context, sessionID string) ([]model.Message, error)
}

type messageRepositoryImpl struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepositoryImpl{db: db}
}

func (repo *messageRepositoryImpl) ListBySessionID(ctx context.Context, sessionID string) ([]model.Message, error) {
	var messages []model.Message
	if err := repo.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("id ASC").
		Find(&messages).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询消息列表失败: %w", err))
	}
	return messages, nil
}

func (repo *messageRepositoryImpl) Create(ctx context.Context, message *model.Message) error {
	if err := repo.db.WithContext(ctx).Create(message).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建消息失败: %w", err))
	}
	return nil
}

func (repo *messageRepositoryImpl) BatchCreate(ctx context.Context, messages []model.Message) error {
	if len(messages) == 0 {
		return nil
	}
	if err := repo.db.WithContext(ctx).Create(&messages).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("批量创建消息失败: %w", err))
	}
	return nil
}

func (repo *messageRepositoryImpl) DeleteBySessionID(ctx context.Context, sessionID string) error {
	if err := repo.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&model.Message{}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("删除消息失败: %w", err))
	}
	return nil
}

func (repo *messageRepositoryImpl) ListBySessionIDExcludeTool(ctx context.Context, sessionID string) ([]model.Message, error) {
	var messages []model.Message
	if err := repo.db.WithContext(ctx).
		Where("session_id = ? AND role != ?", sessionID, "tool").
		Order("id ASC").
		Find(&messages).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询消息列表失败: %w", err))
	}
	return messages, nil
}
