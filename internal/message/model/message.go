package model

import (
	"time"
)

// Message 对应messages表的数据模型
type Message struct {
	ID         int       `json:"id" gorm:"primaryKey;column:id"`
	Title      string    `json:"title" gorm:"type:varchar(255);column:title"`
	Content    string    `json:"content" gorm:"type:mediumtext;column:content"`
	SendTime   time.Time `json:"send_time" gorm:"column:send_time"`
	IsDeleted  string    `json:"is_deleted" gorm:"column:is_deleted;default:N"` // 软删除标志，默认值为N
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
	CreateUser int       `json:"create_user" gorm:"column:create_user"` // 数据创建用户ID
	UpdateUser int       `json:"update_user" gorm:"column:update_user"` // 最后更新数据用户ID
}

// TableName 设置表名
func (*Message) TableName() string {
	return "messages" // 表名指定为messages
}
