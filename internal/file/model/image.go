package model

import (
	"time"
)

// Image 图片数据模型，对应图片表结构
type Image struct {
	ID           int       `json:"id" gorm:"primaryKey;column:id"`                            // 主键
	BizType      string    `json:"biz_type" gorm:"type:varchar(50);column:biz_type"`          // 关联业务类型，ARTICLE-文章，EVENT-活动
	BizID        int       `json:"biz_id" gorm:"column:biz_id"`                               // 关联业务ID
	ObjectName   string    `json:"object_name" gorm:"type:varchar(255);column:object_name"`   // MinIO中的对象名
	URL          string    `json:"url" gorm:"type:varchar(255);column:url"`                   // 图片访问URL
	FileName     string    `json:"file_name" gorm:"type:varchar(255);column:file_name"`       // 原始文件名
	FileSize     int       `json:"file_size" gorm:"column:file_size"`                         // 文件大小（字节）
	ContentType  string    `json:"content_type" gorm:"type:varchar(255);column:content_type"` // 文件MIME类型
	UploadUserID int       `json:"upload_user_id" gorm:"column:upload_user_id"`               // 上传用户ID，关联users表
	CreateTime   time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`      // 创建时间，自动填充
	UpdateTime   time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`      // 更新时间，自动更新
}

// TableName 设置表名
func (*Image) TableName() string {
	return "images"
}
