package model

import (
	"time"
)

// ChatGroupMember 对应chat_group_members表，存储聊天群成员关联数据
type ChatGroupMember struct {
	ID            int64     `json:"id" gorm:"primaryKey;column:id"`                       // 主键ID
	GroupID       int       `json:"group_id" gorm:"column:group_id;not null"`             // 群组id，关联chat_group
	UserID        int       `json:"user_id" gorm:"column:user_id;not null"`               // 用户id
	JoinedAt      time.Time `json:"joined_at" gorm:"column:joined_at;not null"`           // 入群时间
	LastReadMsgID int64     `json:"last_read_msg_id" gorm:"column:last_read_msg_id"`      // 最后已读消息ID，关联chat_messages
	JoinMsgID     int64     `json:"join_msg_id" gorm:"column:join_msg_id"`                // 入群时最大消息ID，关联chat_messages
	CreateTime    time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间
	UpdateTime    time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间
	CreateUser    int       `json:"create_user" gorm:"column:create_user"`                // 数据创建用户ID
	UpdateUser    int       `json:"update_user" gorm:"column:update_user"`                // 最后更新数据用户ID
}

// TableName 绑定模型对应数据库表名
func (*ChatGroupMember) TableName() string {
	return "chat_group_members"
}
