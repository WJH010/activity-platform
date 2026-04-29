package dto

type ListIndustriesResponse struct {
	ID           int    `json:"id"`            // 行业ID
	IndustryCode string `json:"industry_code"` // 行业编码
	IndustryName string `json:"industry_name"` // 行业名称
}

type IndustryUrlID struct {
	ID int `uri:"id" binding:"required"` // 行业ID
}

// CreateIndustryRequest 创建行业请求参数
type CreateIndustryRequest struct {
	IndustryCode string `json:"industry_code" binding:"required,max=50"`  // 行业编码
	IndustryName string `json:"industry_name" binding:"required,max=255"` // 行业名称
}

// UpdateIndustryRequest 更新行业请求参数
type UpdateIndustryRequest struct {
	IndustryCode string `json:"industry_code" binding:"omitempty,non_empty_string,max=50"`  // 行业编码
	IndustryName string `json:"industry_name" binding:"omitempty,non_empty_string,max=255"` // 行业名称
}
