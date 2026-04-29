package repository

import (
	"activity-platform/internal/agent/model"
	"activity-platform/internal/utils"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// SessionRepository 会话数据访问接口
type SessionRepository interface {
	// GetByID 根据ID获取会话
	GetByID(ctx context.Context, id string) (*model.Session, error)
	// ListByUserID 获取用户会话列表
	ListByUserID(ctx context.Context, userID int) ([]model.Session, error)
	// Create 创建会话
	Create(ctx context.Context, session *model.Session) error
	// Update 更新会话
	Update(ctx context.Context, id string, updateFields map[string]interface{}) error
	// SoftDelete 软删除会话
	SoftDelete(ctx context.Context, id string) error
}

type sessionRepositoryImpl struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepositoryImpl{db: db}
}

func (repo *sessionRepositoryImpl) GetByID(ctx context.Context, id string) (*model.Session, error) {
	var session model.Session
	if err := repo.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, utils.DeletedFlagNo).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询会话失败: %w", err))
	}
	return &session, nil
}

func (repo *sessionRepositoryImpl) ListByUserID(ctx context.Context, userID int) ([]model.Session, error) {
	var sessions []model.Session
	if err := repo.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, utils.DeletedFlagNo).
		Order("update_time DESC").
		Find(&sessions).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询会话列表失败: %w", err))
	}
	return sessions, nil
}

func (repo *sessionRepositoryImpl) Create(ctx context.Context, session *model.Session) error {
	if err := repo.db.WithContext(ctx).Create(session).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建会话失败: %w", err))
	}
	return nil
}

func (repo *sessionRepositoryImpl) Update(ctx context.Context, id string, updateFields map[string]interface{}) error {
	result := repo.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ? AND is_deleted = ?", id, utils.DeletedFlagNo).
		Updates(updateFields)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新会话失败: %w", result.Error))
	}
	return nil
}

func (repo *sessionRepositoryImpl) SoftDelete(ctx context.Context, id string) error {
	result := repo.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", id).
		Update("is_deleted", utils.DeletedFlagYes)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("删除会话失败: %w", result.Error))
	}
	return nil
}
