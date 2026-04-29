package dto

// FieldTypeUrlID 用于获取单个领域类型的URL参数
type FieldTypeUrlID struct {
	FieldID int `uri:"id" binding:"required"` // 领域类型ID
}

// CreateFieldTypeRequest 创建领域类型请求参数
type CreateFieldTypeRequest struct {
	FieldCode string `json:"field_code" binding:"required,max=50"`  // 领域编码
	FieldName string `json:"field_name" binding:"required,max=255"` // 领域名称
}

// UpdateFieldTypeRequest 更新领域类型请求参数
type UpdateFieldTypeRequest struct {
	FieldCode string `json:"field_code" binding:"omitempty,non_empty_string,max=50"`  // 领域编码
	FieldName string `json:"field_name" binding:"omitempty,non_empty_string,max=255"` // 领域名称
}

// ListFieldTypesResponse 领域类型列表响应
type ListFieldTypesResponse struct {
	FieldID   int    `json:"field_id"`   // 领域ID
	FieldCode string `json:"field_code"` // 领域编码
	FieldName string `json:"field_name"` // 领域名称
}
