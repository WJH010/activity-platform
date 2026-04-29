package engine

// AgentEventType 引擎事件类型
type AgentEventType string

const (
	EventContent    AgentEventType = "content"
	EventToolCall   AgentEventType = "tool_call"
	EventToolResult AgentEventType = "tool_result"
	EventDone       AgentEventType = "done"
	EventError      AgentEventType = "error"
)

// AgentEvent 引擎输出事件
type AgentEvent struct {
	Type             AgentEventType `json:"type"`
	Content          string         `json:"content,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolName         string         `json:"tool_name,omitempty"`
	ToolArgs         string         `json:"tool_args,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	Result           interface{}    `json:"result,omitempty"`
	Success          bool           `json:"success,omitempty"`
	SessionID        string         `json:"session_id,omitempty"`
	Error            string         `json:"error,omitempty"`
}

// AgentResult 引擎运行完整结果（非流式场景使用）
type AgentResult struct {
	Content         string
	ToolCallRecords []ToolCallRecord
}

// ToolCallRecord 工具调用记录
type ToolCallRecord struct {
	ToolCallID string
	ToolName   string
	Arguments  string
	Result     interface{}
	Success    bool
	ResultMsg  string
}
