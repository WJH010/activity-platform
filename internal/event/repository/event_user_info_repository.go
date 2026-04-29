package repository

import (
	"activity-platform/internal/event/dto"
	"activity-platform/internal/event/model"
	"activity-platform/internal/utils"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type EventUserInfoRepository interface {
	// Create 创建用户信息字段
	Create(ctx context.Context, userField *model.EventUserInfo) error
	// Update 更新用户信息字段
	Update(ctx context.Context, fieldID int, updateFields map[string]interface{}) error
	// List 查询用户信息字段列表
	List(ctx context.Context, req dto.ListEventUserInfoRequest) ([]*model.EventUserInfo, error)
}

// EventUserInfoRepositoryImpl 实现 EventUserInfoRepository 接口
type EventUserInfoRepositoryImpl struct {
	db *gorm.DB
}

// NewEventUserInfoRepository 创建 EventUserInfoRepository 实例
func NewEventUserInfoRepository(db *gorm.DB) EventUserInfoRepository {
	return &EventUserInfoRepositoryImpl{db: db}
}

// Create 创建用户信息字段
func (repo *EventUserInfoRepositoryImpl) Create(ctx context.Context, userField *model.EventUserInfo) error {
	if err := repo.db.WithContext(ctx).Create(userField).Error; err != nil {
		ok, _, _ := utils.IsUniqueConstraintError(err)
		if ok {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "字段已存在")
		}
		return utils.NewSystemError(fmt.Errorf("创建字段失败: %w", err))
	}
	return nil
}

// Update 更新用户信息字段
func (repo *EventUserInfoRepositoryImpl) Update(ctx context.Context, fieldID int, updateFields map[string]interface{}) error {
	// 先检查记录是否存在且未被删除
	var count int64
	if err := repo.db.WithContext(ctx).
		Model(&model.EventUserInfo{}).
		Where("id = ?", fieldID).
		Count(&count).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("检查字段存在性失败: %w", err))
	}

	if count == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "字段不存在或已被删除")
	}

	// 执行更新操作
	result := repo.db.WithContext(ctx).Model(&model.EventUserInfo{}).
		Where("id = ?", fieldID).
		Updates(updateFields)

	err := result.Error
	if err != nil {
		ok, _, _ := utils.IsUniqueConstraintError(err)
		if ok {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "字段已存在")
		}
		return utils.NewSystemError(fmt.Errorf("更新字段失败: %w", err))
	}
	return nil
}

// List 查询用户信息字段列表
func (repo *EventUserInfoRepositoryImpl) List(ctx context.Context, req dto.ListEventUserInfoRequest) ([]*model.EventUserInfo, error) {
	var userFields []*model.EventUserInfo

	query := repo.db.WithContext(ctx)
	if req.Code != "" {
		query = query.Where("code LIKE ?", "%"+req.Code+"%")
	}
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.IsDeleted != "" {
		query = query.Where("is_deleted = ?", req.IsDeleted)
	}
	result := query.Find(&userFields)
	err := result.Error

	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return userFields, nil
}
