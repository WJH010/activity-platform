package repository

import (
	"context"
	"fmt"

	"activity-platform/internal/chat/model"
	"activity-platform/internal/utils"

	"gorm.io/gorm"
)

// ChatRepository 定义聊天数据访问接口
type ChatRepository interface {
	CreateMessage(ctx context.Context, tx *gorm.DB, message *model.ChatMessage) error
	RevokeMessage(ctx context.Context, tx *gorm.DB, messageID int64, senderID int) error
	GetMessageByID(ctx context.Context, messageID int64) (*model.ChatMessage, error)
}

// chatRepositoryImpl 实现接口
type chatRepositoryImpl struct {
	db *gorm.DB
}

// NewChatRepository 创建实例
func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepositoryImpl{db: db}
}

// CreateMessage 在事务中创建一条新的聊天消息
func (repo *chatRepositoryImpl) CreateMessage(ctx context.Context, tx *gorm.DB, message *model.ChatMessage) error {
	if err := tx.WithContext(ctx).Create(message).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建聊天消息失败: %w", err))
	}
	return nil
}

// RevokeMessage 在事务中撤回一条消息
func (repo *chatRepositoryImpl) RevokeMessage(ctx context.Context, tx *gorm.DB, messageID int64, senderID int) error {
	result := tx.WithContext(ctx).Model(&model.ChatMessage{}).
		Where("id = ? AND sender_id = ?", messageID, senderID).
		Update("is_recalled", true)

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("撤回消息失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		// 没有行被影响，可能是消息不存在或无权撤回
		return utils.NewBusinessError(utils.ErrCodePermissionDenied, "消息不存在或您无权撤回")
	}
	return nil
}

// GetMessageByID 根据ID查询消息
func (repo *chatRepositoryImpl) GetMessageByID(ctx context.Context, messageID int64) (*model.ChatMessage, error) {
	var message model.ChatMessage
	if err := repo.db.WithContext(ctx).First(&message, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询消息失败: %w", err))
	}
	return &message, nil
}
