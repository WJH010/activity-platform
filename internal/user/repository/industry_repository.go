package repository

import (
	"context"
	"event-platform/internal/user/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

type IndustryRepository interface {
	// ListIndustries 查询行业列表
	ListIndustries(ctx context.Context) ([]*model.Industries, error)
	// CreateIndustry 创建行业
	CreateIndustry(ctx context.Context, industry *model.Industries) error
	// UpdateIndustry 更新行业信息
	UpdateIndustry(ctx context.Context, industryID int, updateFields map[string]interface{}) error
}

type IndustryRepositoryImpl struct {
	db *gorm.DB // 假设使用 GORM 或其他 ORM 库
}

// NewIndustryRepository 创建行业数据访问实例
func NewIndustryRepository(db *gorm.DB) IndustryRepository {
	return &IndustryRepositoryImpl{db: db}
}

// ListIndustries 查询行业列表
func (repo *IndustryRepositoryImpl) ListIndustries(ctx context.Context) ([]*model.Industries, error) {
	var industries []*model.Industries
	result := repo.db.WithContext(ctx).Find(&industries)
	err := result.Error

	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return industries, nil
}

// CreateIndustry 创建行业
func (repo *IndustryRepositoryImpl) CreateIndustry(ctx context.Context, industry *model.Industries) error {
	// 插入行业数据
	if err := repo.db.WithContext(ctx).Create(industry).Error; err != nil {
		ok, _, _ := utils.IsUniqueConstraintError(err)
		if ok {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "行业已存在")
		}
		return utils.NewSystemError(fmt.Errorf("创建行业失败: %w", err))
	}
	return nil
}

// UpdateIndustry 更新行业信息
func (repo *IndustryRepositoryImpl) UpdateIndustry(ctx context.Context, industryID int, updateFields map[string]interface{}) error {
	// 先检查记录是否存在且未被删除
	var count int64
	if err := repo.db.WithContext(ctx).
		Model(&model.Industries{}).
		Where("id = ? AND is_deleted = ?", industryID, utils.DeletedFlagNo).
		Count(&count).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("检查行业存在性失败: %w", err))
	}

	if count == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "行业不存在或已被删除")
	}

	// 更新行业数据
	result := repo.db.WithContext(ctx).Model(&model.Industries{}).
		Where("id = ?", industryID).Updates(updateFields)

	err := result.Error
	if err != nil {
		ok, _, _ := utils.IsUniqueConstraintError(err)
		if ok {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "行业已存在")
		}
		return utils.NewSystemError(fmt.Errorf("更新行业失败: %w", err))
	}
	return nil
}
