package model

import (
	"time"
)

// FieldType 对应领域表，用于关联查询
type FieldType struct {
	FieldID    int       `json:"field_id" gorm:"primaryKey;column:field_id"`
	FieldCode  string    `json:"field_code" gorm:"column:field_code"`
	FieldName  string    `json:"field_name" gorm:"type:varchar(255);column:field_name"`
	IsDeleted  string    `json:"is_deleted" gorm:"column:is_deleted;default:N"`
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

// TableName 设置表名
func (*FieldType) TableName() string {
	return "field_types"
}
