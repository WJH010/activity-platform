package repository

import (
	"context"
	"event-platform/internal/article/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// FieldTypeRepository 数据访问接口，定义数据访问的方法集
type FieldTypeRepository interface {
	// GetFieldType 获取领域类型列表
	GetFieldType(ctx context.Context) ([]*model.FieldType, error)
	// CreateFieldType 创建领域类型
	CreateFieldType(ctx context.Context, fieldType *model.FieldType) error
	// UpdateFieldType 更新领域类型
	UpdateFieldType(ctx context.Context, fieldID int, updateFields map[string]interface{}) error
}

// FieldTypeRepositoryImpl 实现接口的具体结构体
type FieldTypeRepositoryImpl struct {
	db *gorm.DB
}

// NewFieldTypeRepository 创建数据访问实例
func NewFieldTypeRepository(db *gorm.DB) FieldTypeRepository {
	return &FieldTypeRepositoryImpl{db: db}
}

// GetFieldType 获取领域类型列表
func (repo *FieldTypeRepositoryImpl) GetFieldType(ctx context.Context) ([]*model.FieldType, error) {
	var fieldType []*model.FieldType

	result := repo.db.WithContext(ctx).Find(&fieldType)
	err := result.Error

	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return fieldType, nil
}

// CreateFieldType 创建领域类型
func (repo *FieldTypeRepositoryImpl) CreateFieldType(ctx context.Context, fieldType *model.FieldType) error {
	if err := repo.db.WithContext(ctx).Create(fieldType).Error; err != nil {
		ok, _, _ := utils.IsUniqueConstraintError(err)
		if ok {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "领域类型已存在")
		}
		return utils.NewSystemError(fmt.Errorf("创建领域类型失败: %w", err))
	}
	return nil
}

// UpdateFieldType 更新领域类型
func (repo *FieldTypeRepositoryImpl) UpdateFieldType(ctx context.Context, fieldID int, updateFields map[string]interface{}) error {
	// 先检查记录是否存在且未被删除
	var count int64
	if err := repo.db.WithContext(ctx).
		Model(&model.FieldType{}).
		Where("field_id = ? AND is_deleted = ?", fieldID, utils.DeletedFlagNo).
		Count(&count).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("检查领域类型存在性失败: %w", err))
	}

	if count == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "领域类型不存在或已被删除")
	}

	// 执行更新操作
	result := repo.db.WithContext(ctx).Model(&model.FieldType{}).
		Where("field_id = ?", fieldID).
		Updates(updateFields)

	err := result.Error
	if err != nil {
		ok, _, _ := utils.IsUniqueConstraintError(err)
		if ok {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "领域类型已存在")
		}
		return utils.NewSystemError(fmt.Errorf("更新领域类型失败: %w", err))
	}
	return nil
}
