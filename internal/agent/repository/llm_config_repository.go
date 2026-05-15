package repository

import (
	"context"
	"errors"
	"event-platform/internal/agent/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// LLMConfigRepository LLM配置数据访问接口
type LLMConfigRepository interface {
	// List 获取所有LLM配置
	List(ctx context.Context) ([]model.LLMConfig, error)
	// GetByID 根据ID获取LLM配置
	GetByID(ctx context.Context, id int) (*model.LLMConfig, error)
	// Create 创建LLM配置
	Create(ctx context.Context, config *model.LLMConfig) error
	// Update 更新LLM配置
	Update(ctx context.Context, id int, updateFields map[string]interface{}) error
	// Delete 删除LLM配置
	Delete(ctx context.Context, id int) error
}

type llmConfigRepositoryImpl struct {
	db *gorm.DB
}

func NewLLMConfigRepository(db *gorm.DB) LLMConfigRepository {
	return &llmConfigRepositoryImpl{db: db}
}

func (repo *llmConfigRepositoryImpl) List(ctx context.Context) ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	if err := repo.db.WithContext(ctx).Order("id ASC").Find(&configs).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询LLM配置失败: %w", err))
	}
	return configs, nil
}

func (repo *llmConfigRepositoryImpl) GetByID(ctx context.Context, id int) (*model.LLMConfig, error) {
	var config model.LLMConfig
	if err := repo.db.WithContext(ctx).First(&config, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "LLM配置不存在")
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询LLM配置失败: %w", err))
	}
	return &config, nil
}

func (repo *llmConfigRepositoryImpl) Create(ctx context.Context, config *model.LLMConfig) error {
	if err := repo.db.WithContext(ctx).Create(config).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建LLM配置失败: %w", err))
	}
	return nil
}

func (repo *llmConfigRepositoryImpl) Update(ctx context.Context, id int, updateFields map[string]interface{}) error {
	result := repo.db.WithContext(ctx).Model(&model.LLMConfig{}).Where("id = ?", id).Updates(updateFields)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新LLM配置失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "LLM配置不存在")
	}
	return nil
}

func (repo *llmConfigRepositoryImpl) Delete(ctx context.Context, id int) error {
	result := repo.db.WithContext(ctx).Delete(&model.LLMConfig{}, id)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("删除LLM配置失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "LLM配置不存在")
	}
	return nil
}
