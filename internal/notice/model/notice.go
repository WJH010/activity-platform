package model

import (
	"time"
)

// Notice 公告模型
type Notice struct {
	ID          int        `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Title       string     `json:"title" gorm:"type:varchar(255)"`
	Content     string     `json:"content" gorm:"type:text;not null"`
	ReleaseTime *time.Time `json:"release_time" gorm:"column:release_time"`
	IsDeleted   string     `json:"is_deleted" gorm:"column:is_deleted;default:N"` // 软删除标志，默认值为N
	CreateTime  *time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime  *time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

// TableName 设置表名
func (*Notice) TableName() string {
	return "notices"
}
