package model

import "time"

// EventUserInfo 数据模型
type EventUserInfo struct {
	ID         int       `json:"id" gorm:"primaryKey;column:id"`                                // 主键字段
	Code       string    `json:"code" gorm:"column:code;type:varchar(255)"`                     // 字段编码
	Name       string    `json:"name" gorm:"column:name;type:varchar(255)"`                     // 字段名称
	IsDeleted  string    `json:"is_deleted" gorm:"column:is_deleted;type:varchar(5);default:N"` // 软删除标志，默认值N（varchar类型，长度5）
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`          // 创建时间，GORM自动填充创建时间
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`          // 更新时间，GORM自动填充更新时间
}

// TableName 设置表名
func (*EventUserInfo) TableName() string {
	return "event_user_info"
}
