package model

import "time"

// EventRegistrationInfo 数据模型
type EventRegistrationInfo struct {
	ID          int       `json:"id" gorm:"primaryKey;column:id"`           // 主键
	EventID     int       `json:"event_id" gorm:"column:event_id;not null"` // 活动ID
	UserID      int       `json:"user_id" gorm:"column:user_id;not null"`   // 用户ID
	Name        string    `json:"name" gorm:"column:name;type:varchar(255)"`
	PhoneNumber string    `json:"phone_number" gorm:"column:phone_number;type:varchar(20)"`
	Email       string    `json:"email" gorm:"column:email;type:varchar(64)"`
	Industry    string    `json:"industry" gorm:"column:industry;type:varchar(255)"`
	Position    string    `json:"position" gorm:"column:position;type:varchar(255)"`
	Unit        string    `json:"unit" gorm:"column:unit;type:varchar(255)"`
	Department  string    `json:"department" gorm:"column:department;type:varchar(255)"`
	CreateTime  time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime  time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

// TableName 重写GORM默认表名映射规则
func (*EventRegistrationInfo) TableName() string {
	return "event_registration_info"
}
