package dto

import "time"

// MessageIDRequest 消息内容查询请求参数
type MessageIDRequest struct {
	MessageID int `uri:"id" binding:"required,numeric"` // 消息ID，必须为数字
}

// ListPageRequest 分页请求
type ListPageRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`              // 页码，默认1
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"` // 每页数量，默认10，最大100
}

// HasUnreadMessagesRequest 获取用户是否有未读消息请求参数
type HasUnreadMessagesRequest struct {
	TypeCode string `form:"type_code" binding:"omitempty,user_group_message_type"` // 消息类型代码
}

// ListUserGroupMessageRequest 消息群组列表请求参数
type ListUserGroupMessageRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`                       // 页码，最小为1
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`          // 页大小，1-100
	TypeCode string `form:"type_code" binding:"required,user_group_message_type"` // 消息类型代码
}

// ListMessageByGroupRequest 分页查询分组内消息列表请求参数
type ListMessageByGroupRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`              // 页码，最小为1
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"` // 页大小，1-100
}

// ListMessageByGroupIDRequest 分页查询分组内消息列表请求参数
type ListMessageByGroupIDRequest struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`              // 页码，默认1
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"` // 每页数量，默认10，最大100
	QueryScope string `form:"query_scope" binding:"omitempty,query_scope"` // 查询范围
	Title      string `form:"title" binding:"omitempty,max=255"`           // 消息标题
}

// SendMessageRequest 发送消息请求参数
type SendMessageRequest struct {
	Title   string `json:"title" binding:"required,max=255"` // 消息标题，必填，最大长度255
	Content string `json:"content" binding:"required"`       // 消息内容，必填
}

// MessageContentResponse 消息内容响应结构体
type MessageContentResponse struct {
	ID       int       `json:"id"`
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	SendTime time.Time `json:"send_time"`
}

type MessageGroupDTO struct {
	MsgGroupID     int       `json:"msg_group_id"`
	GroupName      string    `json:"group_name"`
	LatestTitle    string    `json:"latest_title"`
	LatestContent  string    `json:"latest_content"`
	LatestSendTime time.Time `json:"latest_send_time"`
	HasUnread      string    `json:"has_unread"`
	MemberCount    int       `json:"member_count"`
}

type ListMessageDTO struct {
	ID       int       `json:"id"`
	MapID    int       `json:"map_id"` // 关联表 message_group_mappings 的ID
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	SendTime time.Time `json:"send_time"`
}
