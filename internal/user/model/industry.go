package model

import (
	"time"
)

// Industries 对应 news_platform 数据库中 industries 数据表的数据模型
type Industries struct {
	ID           int       `json:"id" gorm:"primaryKey;column:id"`                              // 主键ID
	IndustryCode string    `json:"industry_code" gorm:"not null;size:50;column:industry_code"`  // 行业编码
	IndustryName string    `json:"industry_name" gorm:"not null;size:255;column:industry_name"` // 行业名称
	IsDeleted    string    `json:"is_deleted" gorm:"column:is_deleted;default:N"`               // 软删除标志
	CreateTime   time.Time `json:"create_time" gorm:"column:create_time"`                       // 数据创建时间
	UpdateTime   time.Time `json:"update_time" gorm:"column:update_time"`                       // 数据最后更新时间
}

// TableName 设置当前模型对应的数据库表名
func (*Industries) TableName() string {
	return "industries"
}
