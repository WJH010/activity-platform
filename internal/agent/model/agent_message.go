package model

import "time"

// Message 消息模型
type Message struct {
	ID               int64     `json:"id" gorm:"primaryKey;column:id"`
	SessionID        string    `json:"session_id" gorm:"not null;column:session_id;size:36"`
	Role             string    `json:"role" gorm:"not null;column:role;size:20"`
	Content          string    `json:"content" gorm:"type:mediumtext;column:content"`
	ReasoningContent string    `json:"reasoning_content" gorm:"type:mediumtext;column:reasoning_content"`
	ToolCalls        string    `json:"tool_calls" gorm:"type:text;column:tool_calls"`
	ToolCallID       string    `json:"tool_call_id" gorm:"column:tool_call_id;size:50"`
	SkillName        string    `json:"skill_name" gorm:"column:skill_name;size:100"`
	SkillResult      string    `json:"skill_result" gorm:"type:mediumtext;column:skill_result"`
	CreateTime       time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"` // 数据创建时间，自动生成
	UpdateTime       time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"` // 数据最后更新时间，自动更新
	CreateUser       int       `json:"create_user" gorm:"column:create_user"`                // 创建人ID
	UpdateUser       int       `json:"update_user" gorm:"column:update_user"`
}

func (*Message) TableName() string {
	return "agent_messages"
}
