package model

import (
	"time"
)

// UserMsgGroupMapping 对应 user_msg_group_mappings 表
// 功能说明：作为用户与消息群组的关联中间表，实现多对多关系映射，支持软删除与操作痕迹追溯
type UserMsgGroupMapping struct {
	ID            int       `json:"id" gorm:"primaryKey;column:id"`
	MsgGroupID    int       `json:"msg_group_id" gorm:"not null;column:msg_group_id"`
	UserID        int       `json:"user_id" gorm:"not null;column:user_id"`
	LastReadMsgID int       `json:"last_read_msg_id" gorm:"column:last_read_msg_id;default:0"`     // 用户最后阅读的消息ID，默认值为0
	JoinMsgID     int       `json:"join_msg_id" gorm:"column:join_msg_id;default:0"`               // 用户加入群组时的最新消息ID，默认值为0
	IsDeleted     string    `json:"is_deleted" gorm:"default:N;column:is_deleted;type:varchar(5)"` // 软删除标记：默认 N
	CreateTime    time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime    time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
	CreateUser    int       `json:"create_user" gorm:"column:create_user"`
	UpdateUser    int       `json:"update_user" gorm:"column:update_user"`
}

// TableName 指定模型对应的数据表名为 user_msg_group_mappings
func (*UserMsgGroupMapping) TableName() string {
	return "user_msg_group_mappings"
}
