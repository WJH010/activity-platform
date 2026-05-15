package repository

import (
	"context"
	"errors"
	"event-platform/internal/agent/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// UserLLMPrefRepository 用户LLM偏好数据访问接口
type UserLLMPrefRepository interface {
	// GetByUserID 根据用户ID获取偏好
	GetByUserID(ctx context.Context, userID int) (*model.UserLLMPref, error)
	// Upsert 创建或更新用户偏好
	Upsert(ctx context.Context, pref *model.UserLLMPref) error
}

type userLLMPrefRepositoryImpl struct {
	db *gorm.DB
}

func NewUserLLMPrefRepository(db *gorm.DB) UserLLMPrefRepository {
	return &userLLMPrefRepositoryImpl{db: db}
}

func (repo *userLLMPrefRepositoryImpl) GetByUserID(ctx context.Context, userID int) (*model.UserLLMPref, error) {
	var pref model.UserLLMPref
	if err := repo.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询用户LLM偏好失败: %w", err))
	}
	return &pref, nil
}

func (repo *userLLMPrefRepositoryImpl) Upsert(ctx context.Context, pref *model.UserLLMPref) error {
	var existing model.UserLLMPref
	err := repo.db.WithContext(ctx).Where("user_id = ?", pref.UserID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return utils.NewSystemError(fmt.Errorf("查询用户LLM偏好失败: %w", err))
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建
		if err := repo.db.WithContext(ctx).Create(pref).Error; err != nil {
			return utils.NewSystemError(fmt.Errorf("创建用户LLM偏好失败: %w", err))
		}
	} else {
		// 更新
		if err := repo.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"llm_config_id": pref.LLMConfigID,
		}).Error; err != nil {
			return utils.NewSystemError(fmt.Errorf("更新用户LLM偏好失败: %w", err))
		}
	}
	return nil
}
