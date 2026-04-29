package model

import "time"

// SkillDefinition Skill定义模型
type SkillDefinition struct {
	ID              int       `json:"id" gorm:"primaryKey;column:id"`
	Name            string    `json:"name" gorm:"uniqueIndex;not null;column:name;size:100"`
	DisplayName     string    `json:"display_name" gorm:"not null;column:display_name;size:200"`
	Description     string    `json:"description" gorm:"not null;column:description;type:text"`
	ParamSchema     string    `json:"param_schema" gorm:"type:text;column:param_schema"`
	HttpMethod      string    `json:"http_method" gorm:"column:http_method;size:10"`
	UrlTemplate     string    `json:"url_template" gorm:"type:text;column:url_template"`
	HeadersTemplate string    `json:"headers_template" gorm:"type:text;column:headers_template"`
	BodyTemplate    string    `json:"body_template" gorm:"type:text;column:body_template"`
	AuthRequired    int       `json:"auth_required" gorm:"default:1;column:auth_required"`
	IsEnabled       int       `json:"is_enabled" gorm:"default:1;column:is_enabled"`
	IsBuiltin       int       `json:"is_builtin" gorm:"default:0;column:is_builtin"`
	Category        string    `json:"category" gorm:"column:category;size:50"`
	SortOrder       int       `json:"sort_order" gorm:"default:0;column:sort_order"`
	CreateTime      time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间，自动生成
	UpdateTime      time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间，自动更新
	CreateUser      int       `json:"create_user" gorm:"column:create_user"`                // 创建人ID
	UpdateUser      int       `json:"update_user" gorm:"column:update_user"`
}

func (*SkillDefinition) TableName() string {
	return "agent_skills"
}
