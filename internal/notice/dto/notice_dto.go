package dto

import "time"

// NoticeListRequest 公告列表查询请求参数
type NoticeListRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`              // 页码，最小为1
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"` // 页大小，1-100
}

// NoticeContentRequest 公告内容查询请求参数
type NoticeContentRequest struct {
	ID int `uri:"id" binding:"required,numeric"` // 公告ID，必须为数字
}

// NoticeResponse 公告列表响应结构体
type NoticeResponse struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ReleaseTime time.Time `json:"release_time"`
}

// NoticeContentResponse 公告内容响应结构体
type NoticeContentResponse struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ReleaseTime time.Time `json:"release_time"`
}
