package llm

import "context"

// ProviderType LLM服务提供商类型
type ProviderType string

const (
	ProviderOllama   ProviderType = "ollama"
	ProviderQianwen  ProviderType = "qianwen"
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderDoubao   ProviderType = "doubao"
)

// Message LLM消息结构
type Message struct {
	Role             string     `json:"role"`                        // system/user/assistant/tool
	Content          string     `json:"content"`                     // 消息文本内容
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 推理模型思考过程（DeepSeek等）
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`        // LLM请求调用的工具列表
	ToolCallID       string     `json:"tool_call_id,omitempty"`      // 工具调用结果回传ID
	Name             string     `json:"name,omitempty"`              // 工具名称（role=tool时使用）
}

// ToolCall LLM发起的工具调用
type ToolCall struct {
	ID       string       `json:"id"`       // 工具调用唯一ID
	Type     string       `json:"type"`     // 类型，通常为"function"
	Function FunctionCall `json:"function"` // 函数调用详情
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`      // 函数名
	Arguments string `json:"arguments"` // 参数JSON字符串
}

// ToolDefinition 工具定义（传给LLM的Function Calling描述）
type ToolDefinition struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON Schema对象
}

// LLMResponse LLM完整响应
type LLMResponse struct {
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 推理模型思考过程
	ToolCalls        []ToolCall `json:"tool_calls"`
	FinishReason     string     `json:"finish_reason"` // stop/tool_calls
	Usage            Usage      `json:"usage"`
}

// Usage token使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 推理模型思考过程（增量）
	ToolCalls        []ToolCall `json:"tool_calls"`                  // 增量工具调用
	FinishReason     string     `json:"finish_reason"`
	Done             bool       `json:"done"`
	Usage            *Usage     `json:"usage,omitempty"`
}

// LLMProvider LLM服务提供商统一接口
type LLMProvider interface {
	// Chat 发送消息并获取完整响应
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
	// ChatStream 流式发送消息，返回channel
	ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error)
	// GetModelName 获取当前模型名称
	GetModelName() string
	// GetProviderType 获取提供商类型
	GetProviderType() ProviderType
}
