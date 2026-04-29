package controller

import (
	"activity-platform/internal/article/dto"
	"activity-platform/internal/article/model"
	"activity-platform/internal/article/service"
	"activity-platform/internal/utils"

	"github.com/gin-gonic/gin"
)

// FieldTypeController 控制器
type FieldTypeController struct {
	fieldTypeService service.FieldTypeService
}

// NewFieldTypeController 创建控制器实例
func NewFieldTypeController(fieldTyprService service.FieldTypeService) *FieldTypeController {
	return &FieldTypeController{fieldTypeService: fieldTyprService}
}

// GetFieldType 获取领域类型列表
func (ctr *FieldTypeController) GetFieldType(ctx *gin.Context) {
	// 调用服务层
	fieldTypes, err := ctr.fieldTypeService.GetFieldType(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	var result []dto.ListFieldTypesResponse
	for _, field := range fieldTypes {
		result = append(result, dto.ListFieldTypesResponse{
			FieldID:   field.FieldID,
			FieldCode: field.FieldCode,
			FieldName: field.FieldName,
		})
	}

	// 返回成功响应
	utils.Success(ctx, "success", result)
}

// CreateFieldType 创建领域类型
func (ctr *FieldTypeController) CreateFieldType(ctx *gin.Context) {
	var req dto.CreateFieldTypeRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	fieldType := &model.FieldType{
		FieldCode: req.FieldCode,
		FieldName: req.FieldName,
	}

	if err := ctr.fieldTypeService.CreateFieldType(ctx, fieldType); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// UpdateFieldType 更新领域类型
func (ctr *FieldTypeController) UpdateFieldType(ctx *gin.Context) {
	var urlReq dto.FieldTypeUrlID
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	var req dto.UpdateFieldTypeRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.fieldTypeService.UpdateFieldType(ctx, urlReq.FieldID, req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// DeleteFieldType 删除领域类型
func (ctr *FieldTypeController) DeleteFieldType(ctx *gin.Context) {
	var urlReq dto.FieldTypeUrlID
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	if err := ctr.fieldTypeService.DeleteFieldType(ctx, urlReq.FieldID); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}
