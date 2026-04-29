package dto

import "time"

// CreateGroupReq 创建群组的请求体
type CreateGroupReq struct {
	GroupName string `json:"group_name" binding:"required,min=1,max=32"`
	Desc      string `json:"desc" binding:"max=255"`
}

// GroupInfoResp 返回群组信息的响应体
type GroupInfoResp struct {
	ID        int    `json:"id"`
	GroupName string `json:"group_name"`
	Desc      string `json:"desc"`
	OwnerID   int    `json:"owner_id"`
}

//// AddMembersReq 添加群组成员的请求体
type AddMembersReq struct {
	UserIDs     []int  `json:"user_ids" binding:"required,gt=0"`
	WithHistory string `json:"with_history" binding:"omitempty,oneof=Y N" default:"N"` // 是否附带历史消息 (Y/N)
}

// RemoveMembersReq 移除群组成员的请求体
type RemoveMembersReq struct {
	UserIDs []int `json:"user_ids" binding:"required,gt=0"`
}

// GroupIDReq 从 URL 中获取 group_id 的请求
type GroupIDReq struct {
	GroupID int `uri:"groupId" binding:"required"`
}

// ListGroupMembersReq 查询群成员列表的请求体
type ListGroupMembersReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ListGroupMembersResponse 查询群成员列表的响应体
type ListGroupMembersResponse struct {
	UserID    int    `json:"user_id"`
	Nickname  string `json:"nickname"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// ListUserGroupsResponse 定义了用户群组列表的响应项
type ListUserGroupsResponse struct {
	GroupID           int64      `json:"group_id" example:"1"`             // 群组ID
	GroupName         string     `json:"group_name" example:"技术交流群"`       // 群组名称
	Desc              string     `json:"desc" example:"这是一个技术交流群"`         // 群组描述
	LastMessage       string     `json:"last_message" example:"今晚开会"`      // 最后一条消息内容
	LastMessageSender string     `json:"last_message_sender" example:"张三"` // 最后一条消息发送者
	LastMessageTime   *time.Time `json:"last_message_time"`                // 最后一条消息发送时间
	HasUnread         bool       `json:"has_unread" example:"true"`        // 是否有未读消息
}

// ListGroupMessagesReq 定义了查询群组消息列表的请求参数
type ListGroupMessagesReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	StartDate string `form:"start_date" binding:"omitempty,datetime=2006-01-02"` // 开始日期，格式：YYYY-MM-DD
	EndDate   string `form:"end_date" binding:"omitempty,datetime=2006-01-02"`   // 结束日期，格式：YYYY-MM-DD
	Content   string `form:"content" binding:"omitempty"`                        // 消息内容，模糊查询
}

// ListGroupMessagesResponse 定义了群组消息列表的响应项
type ListGroupMessagesResponse struct {
	MessageID    int64     `json:"message_id"`
	Content      string    `json:"content"`
	SenderID     int       `json:"sender_id"`
	SenderName   string    `json:"sender_name"`
	SenderAvatar string    `json:"sender_avatar"`
	SendAt       time.Time `json:"send_at"`
}

// GetUsersNotInGroupRequest 定义了查询不在群组中用户列表的请求参数
type GetUsersNotInGroupRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=10"`
	Name     string `form:"name"`
}

// NotInGroupUserResponse 定义了查询不在群组中用户列表的响应项
type NotInGroupUserResponse struct {
	UserID       int    `json:"user_id"`
	Nickname     string `json:"nickname"`
	Name         string `json:"name"`
	GenderCode   string `json:"gender_code"`
	Gender       string `json:"gender"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Unit         string `json:"unit"`
	Department   string `json:"department"`
	Position     string `json:"position"`
	Industry     string `json:"industry"`
	IndustryName string `json:"industry_name"`
	Avatar       string `json:"avatar"` //  ListGroupsUsersResponse 中没有，但通常很有用，我们保留它
}

// WebSocketBroadcastMsg 定义了通过 WebSocket 广播给客户端的最终消息结构
type WebSocketBroadcastMsg struct {
	UserID         int        `json:"user_id"`
	SenderNickname string     `json:"sender_nickname"`
	SenderAvatar   string     `json:"sender_avatar"`
	GroupID        int        `json:"group_id"`
	Data           *ClientMsg `json:"data"`
}
