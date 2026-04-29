package model

import (
	"time"
)

// EventUserMapping 对应 event_user_mappings 表的数据模型
type EventUserMapping struct {
	ID         int       `json:"id" gorm:"primaryKey;column:id"`                       // 主键
	UserID     int       `json:"user_id" gorm:"column:user_id"`                        // 用户id，关联users表
	EventID    int       `json:"event_id" gorm:"column:event_id"`                      // 活动id，关联events表
	IsDeleted  string    `json:"is_deleted" gorm:"column:is_deleted;default:N"`        // 软删除标志
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间，自动生成
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间，自动更新
}

// TableName 设置表名
func (*EventUserMapping) TableName() string {
	return "event_user_mappings"
}
