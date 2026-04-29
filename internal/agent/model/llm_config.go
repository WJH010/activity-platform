package model

import "time"

// LLMConfig LLM配置模型
type LLMConfig struct {
	ID           int       `json:"id" gorm:"primaryKey;column:id"`
	ProviderType string    `json:"provider_type" gorm:"not null;column:provider_type"`
	DisplayName  string    `json:"display_name" gorm:"not null;column:display_name"`
	ModelName    string    `json:"model_name" gorm:"not null;column:model_name"`
	ApiUrl       string    `json:"api_url" gorm:"not null;column:api_url"`
	ApiKey       string    `json:"api_key" gorm:"column:api_key"`
	ExtraConfig  string    `json:"extra_config" gorm:"type:text;column:extra_config"`
	IsEnabled    int       `json:"is_enabled" gorm:"default:1;column:is_enabled"`
	CreateTime   time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间，自动生成
	UpdateTime   time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间，自动更新
	CreateUser   int       `json:"create_user" gorm:"column:create_user"`                // 创建人ID
	UpdateUser   int       `json:"update_user" gorm:"column:update_user"`
}

func (*LLMConfig) TableName() string {
	return "agent_llm_configs"
}
