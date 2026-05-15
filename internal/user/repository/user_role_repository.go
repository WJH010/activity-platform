package repository

import (
	"context"
	"event-platform/internal/user/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// UserRoleRepository 定义用户角色仓库接口
type UserRoleRepository interface {
	List(ctx context.Context) ([]*model.UserRole, error)
}

// userRoleRepository 实现 UserRoleRepository 接口
type userRoleRepository struct {
	db *gorm.DB
}

// NewUserRoleRepository 创建用户角色仓库实例
func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{db: db}
}

// List 获取所有用户角色
func (repo *userRoleRepository) List(ctx context.Context) ([]*model.UserRole, error) {
	var roles []*model.UserRole
	if err := repo.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("获取用户角色列表失败: %w", err))
	}
	return roles, nil
}
