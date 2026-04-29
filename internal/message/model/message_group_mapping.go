package model

import (
	"time"
)

// MessageGroupMapping 对应数据库表 message_group_mappings 的数据模型
// 用于存储消息与用户消息群组的关联关系
type MessageGroupMapping struct {
	ID         int       `json:"id" gorm:"primaryKey;column:id"`                                // 主键ID
	MessageID  int       `json:"message_id" gorm:"column:message_id"`                           // 消息id，关联messages表
	MsgGroupID int       `json:"msg_group_id" gorm:"column:msg_group_id"`                       // 群组id，关联user_message_groups表
	IsDeleted  string    `json:"is_deleted" gorm:"type:varchar(5);default:N;column:is_deleted"` // 软删除标志，Y-已删除，N-未删除，默认值N
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`          // 数据创建时间
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`          // 数据最后更新时间
	CreateUser int       `json:"create_user" gorm:"column:create_user"`                         // 数据创建用户ID
	UpdateUser int       `json:"update_user" gorm:"column:update_user"`                         // 最后更新数据用户ID
}

// TableName 设置当前模型对应的数据库表名
func (*MessageGroupMapping) TableName() string {
	return "message_group_mappings"
}
