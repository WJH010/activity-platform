package model

import (
	"time"
)

// ChatGroup 对应chat_groups表，存储聊天群组基础信息
type ChatGroup struct {
	ID              int       `json:"id" gorm:"primaryKey;column:id"`                       // 群ID
	Desc            string    `json:"desc" gorm:"column:desc"`                              // 描述
	GroupName       string    `json:"group_name" gorm:"column:group_name;not null"`         // 群名
	OwnerID         int       `json:"owner_id" gorm:"column:owner_id;not null"`             // 群主
	EventID         int       `json:"event_id" gorm:"column:event_id;default: null"`        // 关联活动ID
	LatestMessageID int64     `json:"latest_message_id" gorm:"column:latest_message_id"`    // 最新消息ID
	CreateTime      time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间
	UpdateTime      time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间
	CreateUser      int       `json:"create_user" gorm:"column:create_user"`                // 数据创建用户ID
	UpdateUser      int       `json:"update_user" gorm:"column:update_user"`                // 最后更新数据用户ID
}

// TableName 绑定当前模型对应的数据表名
func (*ChatGroup) TableName() string {
	return "chat_groups"
}
