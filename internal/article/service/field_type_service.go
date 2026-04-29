package service

import (
	"activity-platform/internal/article/dto"
	"activity-platform/internal/article/model"
	"activity-platform/internal/article/repository"
	"activity-platform/internal/utils"
	"context"
)

// FieldTypeService 服务接口，定义方法，接收 context.Context 和数据模型。
type FieldTypeService interface {
	// GetFieldType 获取领域类型列表
	GetFieldType(ctx context.Context) ([]*model.FieldType, error)
	// CreateFieldType 创建领域类型
	CreateFieldType(ctx context.Context, fieldType *model.FieldType) error
	// UpdateFieldType 更新领域类型
	UpdateFieldType(ctx context.Context, fieldID int, req dto.UpdateFieldTypeRequest) error
	// DeleteFieldType 删除领域类型
	DeleteFieldType(ctx context.Context, fieldID int) error
}

// FieldTypeServiceImpl 实现接口的具体结构体，持有数据访问层接口 Repository 的实例
type FieldTypeServiceImpl struct {
	fieldTypeRepo repository.FieldTypeRepository
}

// NewFieldTypeService 创建服务实例
func NewFieldTypeService(fieldTypeRepo repository.FieldTypeRepository) FieldTypeService {
	return &FieldTypeServiceImpl{fieldTypeRepo: fieldTypeRepo}
}

// GetFieldType 获取领域类型列表
func (svc *FieldTypeServiceImpl) GetFieldType(ctx context.Context) ([]*model.FieldType, error) {
	return svc.fieldTypeRepo.GetFieldType(ctx)
}

// CreateFieldType 创建领域类型
func (svc *FieldTypeServiceImpl) CreateFieldType(ctx context.Context, fieldType *model.FieldType) error {
	return svc.fieldTypeRepo.CreateFieldType(ctx, fieldType)
}

// UpdateFieldType 更新领域类型
func (svc *FieldTypeServiceImpl) UpdateFieldType(ctx context.Context, fieldID int, req dto.UpdateFieldTypeRequest) error {
	updateFields := make(map[string]interface{})
	if req.FieldCode != "" {
		updateFields["field_code"] = req.FieldCode
	}
	if req.FieldName != "" {
		updateFields["field_name"] = req.FieldName
	}

	if len(updateFields) == 0 {
		return nil
	}

	return svc.fieldTypeRepo.UpdateFieldType(ctx, fieldID, updateFields)
}

// DeleteFieldType 删除领域类型(通过更新is_deleted字段实现)
func (svc *FieldTypeServiceImpl) DeleteFieldType(ctx context.Context, fieldID int) error {
	// 复用更新方法，设置is_deleted标志
	updateFields := map[string]interface{}{
		"is_deleted": utils.DeletedFlagYes,
	}
	return svc.fieldTypeRepo.UpdateFieldType(ctx, fieldID, updateFields)
}
