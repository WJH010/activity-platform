package dto

import "time"

// EventListRequest 活动列表查询请求参数
type EventListRequest struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`              // 页码，最小为1
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"` // 页大小，1-100
	EventStatus string `form:"event_status" binding:"omitempty"`            // 活动状态
	QueryScope  string `form:"query_scope" binding:"omitempty,query_scope"` // 查询范围，默认只查询未删除数据
	EventTitle  string `form:"event_title" binding:"omitempty"`             // 活动标题
}

// EventDetailRequest 活动详情查询请求参数
type EventDetailRequest struct {
	EventID int `uri:"id" binding:"required,numeric"` // 活动ID，必须为数字
}

// EventRegistrationRequest 活动报名请求参数
type EventRegistrationRequest struct {
	EventID int `json:"event_id" binding:"required,numeric"` // 活动ID
}

// CreateEventRequest 创建活动请求参数
type CreateEventRequest struct {
	Title                 string  `json:"title" binding:"required,max=255"`                       // 活动标题
	Detail                string  `json:"detail" binding:"required"`                              // 活动内容
	EventStartTime        string  `json:"event_start_time" binding:"required,time_format"`        // 活动开始时间
	EventEndTime          string  `json:"event_end_time" binding:"required,time_format"`          // 活动结束时间
	RegistrationStartTime string  `json:"registration_start_time" binding:"required,time_format"` // 活动报名开始时间
	RegistrationEndTime   string  `json:"registration_end_time" binding:"required,time_format"`   // 活动报名截止时间
	EventAddress          string  `json:"event_address" binding:"required,max=255"`               // 活动地址
	RegistrationFee       float64 `json:"registration_fee" binding:"gte=0"`                       // 报名费用，必须大于或等于 0
	CoverImageURL         string  `json:"cover_image_url" binding:"omitempty,url"`                // 封面图片URL
	ImageIDList           []int   `json:"image_id_list" binding:"omitempty,dive,min=1"`           // 图片ID列表
	UserInfoIDList        []int   `json:"user_info_id_list" binding:"omitempty,dive,min=1"`       // 所需用户信息ID列表
}

// UpdateEventRequest 更新活动请求参数
type UpdateEventRequest struct {
	Title                 *string  `json:"title" binding:"omitempty,non_empty_string,max=255"`                       // 活动标题
	Detail                *string  `json:"detail" binding:"omitempty,non_empty_string"`                              // 活动内容
	EventStartTime        *string  `json:"event_start_time" binding:"omitempty,non_empty_string,time_format"`        // 活动开始时间
	EventEndTime          *string  `json:"event_end_time" binding:"omitempty,non_empty_string,time_format"`          // 活动结束时间
	RegistrationStartTime *string  `json:"registration_start_time" binding:"omitempty,non_empty_string,time_format"` // 活动报名开始时间
	RegistrationEndTime   *string  `json:"registration_end_time" binding:"omitempty,non_empty_string,time_format"`   // 活动报名截止时间
	EventAddress          *string  `json:"event_address" binding:"omitempty,non_empty_string,max=255"`               // 活动地址
	RegistrationFee       *float64 `json:"registration_fee" binding:"omitempty,gte=0"`                               // 报名费用，必须大于或等于 0
	CoverImageURL         *string  `json:"cover_image_url" binding:"omitempty,url"`                                  // 封面图片URL
	GroupID               *int     `json:"group_id" binding:"omitempty,numeric"`                                     // 关联的消息群组ID
	ImageIDList           *[]int   `json:"image_id_list" binding:"omitempty,dive,min=1"`                             // 图片ID列表
}

// EventListResponse 活动列表响应结构体
type EventListResponse struct {
	ID                    int       `json:"id"`                      // 活动ID
	Title                 string    `json:"title"`                   // 活动标题
	EventStartTime        time.Time `json:"event_start_time"`        // 活动开始时间
	EventEndTime          time.Time `json:"event_end_time"`          // 活动结束时间
	RegistrationStartTime time.Time `json:"registration_start_time"` // 活动报名开始时间
	RegistrationEndTime   time.Time `json:"registration_end_time"`   // 活动报名截止时间
	EventAddress          string    `json:"event_address"`           // 活动地址
	RegistrationFee       float64   `json:"registration_fee"`        // 报名费用
	Status                string    `json:"status"`                  // 活动状态
	CoverImageURL         string    `json:"cover_image_url"`         // 封面图片URL
	MemberCount           int       `json:"member_count"`            // 报名人数
	GroupID               int       `json:"group_id"`                // 关联的消息群组ID
}

// Image 关联图片列表结构体
type Image struct {
	ImageID int    `json:"image_id"`
	URL     string `json:"url"`
}

// EventUserInfo 关联用户信息字段结构体
type EventUserInfo struct {
	UserInfoID int    `json:"user_info_id"` // 用户信息ID
	Code       string `json:"code"`         // 用户信息字段编码
	Name       string `json:"name"`         // 用户信息字段名称
}

// EventDetailResponse 活动详情响应结构体
type EventDetailResponse struct {
	Title                 string          `json:"title"`                   // 活动标题
	Detail                string          `json:"detail"`                  // 活动内容
	EventStartTime        time.Time       `json:"event_start_time"`        // 活动开始时间
	EventEndTime          time.Time       `json:"event_end_time"`          // 活动结束时间
	RegistrationStartTime time.Time       `json:"registration_start_time"` // 活动报名开始时间
	RegistrationEndTime   time.Time       `json:"registration_end_time"`   // 活动报名截止时间
	EventAddress          string          `json:"event_address"`           // 活动地址
	RegistrationFee       float64         `json:"registration_fee"`        // 报名费用
	Status                string          `json:"status"`                  // 活动状态
	CoverImageURL         string          `json:"cover_image_url"`         // 封面图片URL
	GroupID               int             `json:"group_id"`                // 关联的消息群组ID
	Images                []Image         `json:"images"`                  // 图片列表
	UserInfo              []EventUserInfo `json:"user_info"`               // 用户信息字段列表
}

// ListEventRegUserResponse 活动报名列表查询请求参数
type ListEventRegUserResponse struct {
	Name         string `json:"name"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Industry     string `json:"industry"`
	IndustryName string `json:"industry_name"`
	Position     string `json:"position"`
	Unit         string `json:"unit"`
	Department   string `json:"department"`
}

// EventUpdatedSinceRequest 增量同步查询请求参数
type EventUpdatedSinceRequest struct {
	Since    time.Time `form:"since" binding:"required"`
	PageSize int       `form:"page_size" binding:"omitempty,min=1,max=200"`
	Page     int       `form:"page" binding:"omitempty,min=1"`
}

// EventUpdatedSinceResponse 增量同步查询响应结构体
// 包含详情内容，避免N+1查询；包含is_deleted和update_time，用于增量同步判断
type EventUpdatedSinceResponse struct {
	ID                    int       `json:"id"`
	Title                 string    `json:"title"`
	Detail                string    `json:"detail"`
	EventStartTime        time.Time `json:"event_start_time"`
	EventEndTime          time.Time `json:"event_end_time"`
	RegistrationStartTime time.Time `json:"registration_start_time"`
	RegistrationEndTime   time.Time `json:"registration_end_time"`
	EventAddress          string    `json:"event_address"`
	IsDeleted             string    `json:"is_deleted"`
	UpdateTime            time.Time `json:"update_time"`
}
