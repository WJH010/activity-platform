package model

import "time"

// Session 会话模型
type Session struct {
	ID          string    `json:"id" gorm:"primaryKey;column:id;size:36"`
	UserID      int       `json:"user_id" gorm:"not null;column:user_id"`
	Title       string    `json:"title" gorm:"column:title;size:200"`
	LLMConfigID *int      `json:"llm_config_id" gorm:"column:llm_config_id"`
	IsDeleted   string    `json:"is_deleted" gorm:"column:is_deleted;default:N"`
	CreateTime  time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间，自动生成
	UpdateTime  time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间，自动更新
	CreateUser  int       `json:"create_user" gorm:"column:create_user"`                // 创建人ID
	UpdateUser  int       `json:"update_user" gorm:"column:update_user"`
}

func (*Session) TableName() string {
	return "agent_sessions"
}
