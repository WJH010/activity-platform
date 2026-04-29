package service

import (
	"activity-platform/internal/user/dto"
	"activity-platform/internal/user/repository"
	"context"
)

// UserRoleService 定义用户角色服务接口
type UserRoleService interface {
	List(ctx context.Context) ([]*dto.UserRoleListDTO, error)
}

// userRoleService 实现 UserRoleService 接口
type userRoleService struct {
	userRoleRepo repository.UserRoleRepository
}

// NewUserRoleService 创建用户角色服务实例
func NewUserRoleService(userRoleRepo repository.UserRoleRepository) UserRoleService {
	return &userRoleService{userRoleRepo: userRoleRepo}
}

// List 获取所有用户角色
func (svc *userRoleService) List(ctx context.Context) ([]*dto.UserRoleListDTO, error) {
	roles, err := svc.userRoleRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	var roleDTOs []*dto.UserRoleListDTO
	for _, role := range roles {
		roleDTOs = append(roleDTOs, &dto.UserRoleListDTO{
			RoleCode: role.RoleCode,
			RoleName: role.RoleName,
		})
	}
	return roleDTOs, nil
}
