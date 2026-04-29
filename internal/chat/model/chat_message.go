package model

import (
	"time"
)

// ChatMessage 对应chat_messages表，存储实时聊天消息
type ChatMessage struct {
	ID         int64     `json:"id" gorm:"primaryKey;column:id"`
	GroupID    int       `json:"group_id" gorm:"column:group_id;not null"`
	SenderID   int       `json:"sender_id" gorm:"column:sender_id;not null"`
	SendAt     time.Time `json:"send_at" gorm:"column:send_at;not null"`    // 发送时间
	MsgType    int       `json:"msg_type" gorm:"column:msg_type;default:1"` // 消息类型：1-文本 2-系统通知 3-撤回通知
	Content    string    `json:"content" gorm:"column:content;type:text;not null"`
	IsRecalled int       `json:"is_recalled" gorm:"column:is_recalled;default:0"` // 撤回标志, 0-否，1-是
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
	CreateUser int       `json:"create_user" gorm:"column:create_user"`
	UpdateUser int       `json:"update_user" gorm:"column:update_user"`
}

// TableName 设置表名
func (*ChatMessage) TableName() string {
	return "chat_messages"
}
