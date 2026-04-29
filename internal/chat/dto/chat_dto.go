package dto

import (
	"encoding/json"
	"time"
)

// WebSocketMsg 是服务器内部处理和广播消息时使用的数据结构。
type WebSocketMsg struct {
	UserID  int             `json:"user_id"`  // 发送者ID（由服务器填充）
	GroupID int             `json:"group_id"` // 群组ID（由服务器填充）
	Data    json.RawMessage `json:"data"`     // 客户端发送的原始消息数据
}

// ClientMsg 是客户端发送到服务器的原始消息的预期结构。
// 我们在 Service 层解析 Data 字段时会用到它。
type ClientMsg struct {
	ID           int64     `json:"id,omitempty"`  // 消息的唯一ID，创建后由后端填充
	Type         string    `json:"type"`          // 消息类型，例如 "chat", "recall"
	Content      string    `json:"content"`       // 消息的具体内容
	SendAt       time.Time `json:"send_at"`       // 消息发送者发送时间
	SenderName   string    `json:"sender_name"`   // 消息发送者的昵称
	SenderAvatar string    `json:"sender_avatar"` // 消息发送者的头像
}
