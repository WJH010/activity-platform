package dto

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID string `json:"session_id"`                 // 可选，已有会话ID
	Content   string `json:"content" binding:"required"` // 用户消息内容
}

// StreamChunk SSE流式响应块
type StreamChunk struct {
	Type      string      `json:"type"`       // content/tool_call/tool_result/done/error
	Content   string      `json:"content"`    // 文本内容（type=content时）
	ToolName  string      `json:"tool_name"`  // 工具名（type=tool_call/tool_result时）
	Status    string      `json:"status"`     // 执行状态（type=tool_call时：executing）
	Result    interface{} `json:"result"`     // 工具结果（type=tool_result时）
	SessionID string      `json:"session_id"` // 会话ID（type=done时）
}

// SessionResponse 会话响应
type SessionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// MessageResponse 消息响应
type MessageResponse struct {
	ID          int64       `json:"id"`
	Role        string      `json:"role"`
	Content     string      `json:"content"`
	ToolCalls   interface{} `json:"tool_calls,omitempty"`
	SkillName   string      `json:"skill_name,omitempty"`
	SkillResult interface{} `json:"skill_result,omitempty"`
	CreatedAt   string      `json:"created_at"`
}
