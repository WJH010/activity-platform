package dto

// CreateEventUserInfoRequest 创建用户字段请求参数
type CreateEventUserInfoRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// UpdateEventUserInfoRequest 更新用户字段请求参数
type UpdateEventUserInfoRequest struct {
	Name string `json:"name" binding:"required"`
}

// ListEventUserInfoRequest 查询用户字段列表请求参数
type ListEventUserInfoRequest struct {
	Code      string `form:"code"`
	Name      string `form:"name"`
	IsDeleted string `form:"is_deleted"`
}

type UpdateEventUserInfoStatusRequest struct {
	IsDeleted string `json:"is_deleted" binding:"required"`
}

// ListEventUserInfoResponse 查询用户字段列表响应参数
type ListEventUserInfoResponse struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	IsDeleted     string `json:"is_deleted"`
	IsDeletedDesc string `json:"is_deleted_desc"`
}

// GetEventUserInfoResponse 查询用户字段详情响应参数
type GetEventUserInfoResponse struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	IsDeleted     string `json:"is_deleted"`
	IsDeletedDesc string `json:"is_deleted_desc"`
}

// EventUserInfoIDRequest 用户字段ID请求参数
type EventUserInfoIDRequest struct {
	ID int `uri:"id"`
}
