package dto

import "mime/multipart"

// FileUploadRequest 文件上传请求参数
type FileUploadRequest struct {
	BizType string                `form:"biz_type" binding:"omitempty"` // 业务类型，ARTICLE-文章，EVENT-活动等
	BizID   int                   `form:"biz_id" binding:"omitempty,numeric"`
	File    *multipart.FileHeader `form:"file" binding:"required"` // 文件字段，必填
}

// FileDeleteRequest 文件删除请求参数
type FileDeleteRequest struct {
	ID int `uri:"id" binding:"required,numeric"` // 图片ID，必填
}

type FileUploadResponse struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
	URL      string `json:"url"`
}
