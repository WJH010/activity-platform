package model

import "time"

// UserLLMPref 用户LLM偏好模型
type UserLLMPref struct {
	ID          int       `json:"id" gorm:"primaryKey;column:id"`
	UserID      int       `json:"user_id" gorm:"not null;column:user_id"`
	LLMConfigID int       `json:"llm_config_id" gorm:"not null;column:llm_config_id"`
	IsDefault   int       `json:"is_default" gorm:"default:1;column:is_default"`
	CreateTime  time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间，自动生成
	UpdateTime  time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间，自动更新
	CreateUser  int       `json:"create_user" gorm:"column:create_user"`                // 创建人ID
	UpdateUser  int       `json:"update_user" gorm:"column:update_user"`                // 更新人ID

}

func (*UserLLMPref) TableName() string {
	return "agent_user_llm_prefs"
}
