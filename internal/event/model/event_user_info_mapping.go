package model

import "time"

// EventUserInfoMapping 数据模型
type EventUserInfoMapping struct {
	ID         int       `json:"id" gorm:"primaryKey;column:id"`                       // 主键
	EventID    int       `json:"event_id" gorm:"column:event_id"`                      // 活动ID，关联事件表主键
	UserInfoID int       `json:"user_info_id" gorm:"column:user_info_id"`              // 用户信息ID，关联用户信息表主键
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 创建时间，GORM自动填充记录创建时的时间
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 更新时间，GORM自动填充记录更新时的时间
}

// TableName 指定表名
func (*EventUserInfoMapping) TableName() string {
	return "event_user_info_mappings"
}
