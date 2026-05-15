package repository

import (
	"context"
	"errors"
	"event-platform/internal/agent/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// SkillRepository Skill定义数据访问接口
type SkillRepository interface {
	// List 获取所有Skill定义
	List(ctx context.Context) ([]model.SkillDefinition, error)
	// GetByID 根据ID获取Skill定义
	GetByID(ctx context.Context, id int) (*model.SkillDefinition, error)
	// GetByName 根据名称获取Skill定义
	GetByName(ctx context.Context, name string) (*model.SkillDefinition, error)
	// ListEnabled 获取所有启用的Skill定义
	ListEnabled(ctx context.Context) ([]model.SkillDefinition, error)
	// Create 创建Skill定义
	Create(ctx context.Context, skill *model.SkillDefinition) error
	// Update 更新Skill定义
	Update(ctx context.Context, id int, updateFields map[string]interface{}) error
	// Delete 删除Skill定义
	Delete(ctx context.Context, id int) error
}

type skillRepositoryImpl struct {
	db *gorm.DB
}

func NewSkillRepository(db *gorm.DB) SkillRepository {
	return &skillRepositoryImpl{db: db}
}

func (repo *skillRepositoryImpl) List(ctx context.Context) ([]model.SkillDefinition, error) {
	var skills []model.SkillDefinition
	if err := repo.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&skills).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询Skill列表失败: %w", err))
	}
	return skills, nil
}

func (repo *skillRepositoryImpl) GetByID(ctx context.Context, id int) (*model.SkillDefinition, error) {
	var skill model.SkillDefinition
	if err := repo.db.WithContext(ctx).First(&skill, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "Skill不存在")
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询Skill失败: %w", err))
	}
	return &skill, nil
}

func (repo *skillRepositoryImpl) GetByName(ctx context.Context, name string) (*model.SkillDefinition, error) {
	var skill model.SkillDefinition
	if err := repo.db.WithContext(ctx).Where("name = ?", name).First(&skill).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询Skill失败: %w", err))
	}
	return &skill, nil
}

func (repo *skillRepositoryImpl) ListEnabled(ctx context.Context) ([]model.SkillDefinition, error) {
	var skills []model.SkillDefinition
	if err := repo.db.WithContext(ctx).
		Where("is_enabled = ?", 1).
		Order("sort_order ASC, id ASC").
		Find(&skills).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询启用的Skill列表失败: %w", err))
	}
	return skills, nil
}

func (repo *skillRepositoryImpl) Create(ctx context.Context, skill *model.SkillDefinition) error {
	if err := repo.db.WithContext(ctx).Create(skill).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建Skill失败: %w", err))
	}
	return nil
}

func (repo *skillRepositoryImpl) Update(ctx context.Context, id int, updateFields map[string]interface{}) error {
	result := repo.db.WithContext(ctx).Model(&model.SkillDefinition{}).Where("id = ?", id).Updates(updateFields)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新Skill失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "Skill不存在")
	}
	return nil
}

func (repo *skillRepositoryImpl) Delete(ctx context.Context, id int) error {
	result := repo.db.WithContext(ctx).Delete(&model.SkillDefinition{}, id)
	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("删除Skill失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "Skill不存在")
	}
	return nil
}
