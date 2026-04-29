package service

import (
	"activity-platform/internal/event/dto"
	"activity-platform/internal/event/model"
	"activity-platform/internal/event/repository"
	"context"
)

type EventUserInfoService interface {
	// Create 创建用户信息字段
	Create(ctx context.Context, eventUserInfo *model.EventUserInfo) error
	// Update 更新用户信息字段
	Update(ctx context.Context, fieldID int, req dto.UpdateEventUserInfoRequest) error
	// List 查询用户信息字段列表
	List(ctx context.Context, req dto.ListEventUserInfoRequest) ([]*dto.ListEventUserInfoResponse, error)
	// Delete 删除用户信息字段（软删除）
	Delete(ctx context.Context, fieldID int) error
	// Restore 恢复用户信息字段（软删除）
	Restore(ctx context.Context, fieldID int) error
}

// EventUserInfoServiceImpl 实现 EventUserInfoService 接口
type EventUserInfoServiceImpl struct {
	eventUserInfoRepo repository.EventUserInfoRepository
}

// NewEventUserInfoService 创建 EventUserInfoService 实例
func NewEventUserInfoService(eventUserInfoRepo repository.EventUserInfoRepository) EventUserInfoService {
	return &EventUserInfoServiceImpl{eventUserInfoRepo: eventUserInfoRepo}
}

// Create 创建用户信息字段
func (svc *EventUserInfoServiceImpl) Create(ctx context.Context, eventUserInfo *model.EventUserInfo) error {
	return svc.eventUserInfoRepo.Create(ctx, eventUserInfo)
}

// Update 更新用户信息字段
func (svc *EventUserInfoServiceImpl) Update(ctx context.Context, fieldID int, req dto.UpdateEventUserInfoRequest) error {
	updateFields := make(map[string]interface{})
	if req.Name != "" {
		updateFields["name"] = req.Name
	}

	if len(updateFields) == 0 {
		return nil
	}
	return svc.eventUserInfoRepo.Update(ctx, fieldID, updateFields)
}

// List 查询用户信息字段列表
func (svc *EventUserInfoServiceImpl) List(ctx context.Context, req dto.ListEventUserInfoRequest) ([]*dto.ListEventUserInfoResponse, error) {
	eventUserInfos, err := svc.eventUserInfoRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	response := make([]*dto.ListEventUserInfoResponse, 0)
	for _, eventUserInfo := range eventUserInfos {
		response = append(response, &dto.ListEventUserInfoResponse{
			ID:            eventUserInfo.ID,
			Code:          eventUserInfo.Code,
			Name:          eventUserInfo.Name,
			IsDeleted:     eventUserInfo.IsDeleted,
			IsDeletedDesc: map[string]string{"N": "正常", "Y": "已删除"}[eventUserInfo.IsDeleted],
		})
	}
	return response, nil
}

// Delete 删除用户信息字段（软删除）
func (svc *EventUserInfoServiceImpl) Delete(ctx context.Context, fieldID int) error {
	updateFields := make(map[string]interface{})
	updateFields["is_deleted"] = "Y"
	return svc.eventUserInfoRepo.Update(ctx, fieldID, updateFields)
}

// Restore 恢复用户信息字段（软删除）
func (svc *EventUserInfoServiceImpl) Restore(ctx context.Context, fieldID int) error {
	updateFields := make(map[string]interface{})
	updateFields["is_deleted"] = "N"
	return svc.eventUserInfoRepo.Update(ctx, fieldID, updateFields)
}
