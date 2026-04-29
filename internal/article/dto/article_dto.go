package dto

import (
	"time"
)

// ArticleListRequest 文章列表查询请求参数
type ArticleListRequest struct {
	Page         int    `form:"page" binding:"omitempty,min=1"`               // 页码，最小为1
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`  // 页大小，1-100
	ArticleTitle string `form:"article_title"`                                // 文章标题
	FieldType    string `form:"field_type" binding:"omitempty"`               // 领域类型代码
	IsSelection  int    `form:"is_selection" binding:"omitempty,numeric"`     // 是否精选，必须为数字
	ArticleType  string `form:"article_type"`                                 // 文章类型
	ReleaseTime  string `form:"release_time" binding:"omitempty,time_format"` // 发布时间
	QueryScope   string `form:"query_scope" binding:"omitempty,query_scope"`  // 查询范围
}

// ArticleContentRequest 文章内容查询请求参数
type ArticleContentRequest struct {
	ArticleID int `uri:"id" binding:"required,numeric"` // 文章ID，必须为数字
}

// CreateArticleRequest 创建文章请求参数
type CreateArticleRequest struct {
	ArticleTitle   string `json:"article_title" binding:"required,max=255"` // 文章标题
	ArticleType    string `json:"article_type" binding:"required"`          // 文章
	BriefContent   string `json:"brief_content"`
	ArticleContent string `json:"article_content" binding:"required"`
	IsSelection    int    `json:"is_selection" binding:"oneof=0 1"` // 默认=2，1：精选，2：非精选
	FieldType      string `json:"field_type"`
	CoverImageURL  string `json:"cover_image_url"` // 封面图片URL
	ArticleSource  string `json:"article_source"`  // 文章来源
	ImageIDList    []int  `json:"image_id_list"`   // 关联图片ID列表
}

// UpdateArticleRequest 更新文章请求参数
type UpdateArticleRequest struct {
	ArticleTitle   *string `json:"article_title" binding:"omitempty,non_empty_string,max=255"`
	ArticleType    *string `json:"article_type" binding:"omitempty,non_empty_string"`    // 文章类型
	BriefContent   *string `json:"brief_content" binding:"omitempty"`                    // 摘要
	ArticleContent *string `json:"article_content" binding:"omitempty,non_empty_string"` // 文章内容
	IsSelection    *int    `json:"is_selection" binding:"omitempty,numeric,oneof=0 1"`   // 是否精选
	FieldType      *string `json:"field_type" binding:"omitempty"`                       // 领域类型
	CoverImageURL  *string `json:"cover_image_url" binding:"omitempty,url"`              // 封面图URL
	ArticleSource  *string `json:"article_source" binding:"omitempty"`                   // 文章来源
	ImageIDList    *[]int  `json:"image_id_list" binding:"omitempty,dive,min=1"`         // 图片ID列表
}

// ArticleListResponse 文章列表响应结构体
type ArticleListResponse struct {
	ArticleID       int       `json:"article_id"`
	ArticleTitle    string    `json:"article_title"`
	ArticleTypeCode string    `json:"article_type_code"`
	ArticleType     string    `json:"article_type"`
	FieldName       string    `json:"field_name"`
	ReleaseTime     time.Time `json:"release_time"`
	BriefContent    string    `json:"brief_content"`
	IsSelection     int       `json:"is_selection"`
	CoverImageURL   string    `json:"cover_image_url"`
	ArticleSource   string    `json:"article_source"`
}

// Image 关联图片列表结构体
type Image struct {
	ImageID int    `json:"image_id"`
	URL     string `json:"url"`
}

// ArticleContentResponse 文章内容响应结构体
type ArticleContentDTO struct {
	ArticleID       int       `json:"article_id"`
	ArticleTitle    string    `json:"article_title"`
	BriefContent    string    `json:"brief_content"`
	FieldType       string    `json:"field_type"`
	FieldName       string    `json:"field_name"`
	ReleaseTime     time.Time `json:"release_time"`
	ArticleContent  string    `json:"article_content"`
	ArticleTypeCode string    `json:"article_type_code"`
	ArticleType     string    `json:"article_type"`
	ArticleSource   string    `json:"article_source"`
	IsSelection     int       `json:"is_selection"`
	CoverImageURL   string    `json:"cover_image_url"`
}

// ArticleContentResponse 文章内容响应结构体
type ArticleContentResponse struct {
	ArticleID       int       `json:"article_id"`
	ArticleTitle    string    `json:"article_title"`
	BriefContent    string    `json:"brief_content"`
	FieldType       string    `json:"field_type"`
	FieldName       string    `json:"field_name"`
	ReleaseTime     time.Time `json:"release_time"`
	ArticleContent  string    `json:"article_content"`
	ArticleTypeCode string    `json:"article_type_code"`
	ArticleType     string    `json:"article_type"`
	ArticleSource   string    `json:"article_source"`
	IsSelection     int       `json:"is_selection"`
	CoverImageURL   string    `json:"cover_image_url"`
	Images          []Image   `json:"images"`
}

// ArticleUpdatedSinceRequest 增量同步查询请求参数
type ArticleUpdatedSinceRequest struct {
	Since    time.Time `form:"since" binding:"required"`
	PageSize int       `form:"page_size" binding:"omitempty,min=1,max=200"`
	Page     int       `form:"page" binding:"omitempty,min=1"`
}

// ArticleUpdatedSinceResponse 增量同步查询响应结构体
// 包含正文内容，避免N+1查询；包含is_deleted和update_time，用于增量同步判断
type ArticleUpdatedSinceResponse struct {
	ArticleID      int       `json:"article_id"`
	ArticleTitle   string    `json:"article_title"`
	ArticleContent string    `json:"article_content"`
	ArticleType    string    `json:"article_type_code"`
	FieldType      string    `json:"field_type"`
	BriefContent   string    `json:"brief_content"`
	IsSelection    int       `json:"is_selection"`
	ReleaseTime    time.Time `json:"release_time"`
	IsDeleted      string    `json:"is_deleted"`
	UpdateTime     time.Time `json:"update_time"`
}
