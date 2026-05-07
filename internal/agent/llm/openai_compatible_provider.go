package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// OpenAICompatibleProvider OpenAI兼容API的通用Provider
// 适用于 Ollama、千问、DeepSeek、豆包等兼容OpenAI格式的服务
type OpenAICompatibleProvider struct {
	providerType ProviderType
	baseURL      string
	apiKey       string
	modelName    string
	httpClient   *http.Client
}

// NewOpenAICompatibleProvider 创建OpenAI兼容Provider
func NewOpenAICompatibleProvider(providerType ProviderType, baseURL, apiKey, modelName string) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		providerType: providerType,
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiKey:       apiKey,
		modelName:    modelName,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM响应可能较慢
		},
	}
}

// OpenAI API 请求/响应结构体

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []chatTool    `json:"tools,omitempty"`
	Stream   bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	Name             string         `json:"name,omitempty"`
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponse struct {
	ID      string       `json:"id"`
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SSE 流式响应结构体

type streamResponse struct {
	ID      string         `json:"id"`
	Choices []streamChoice `json:"choices"`
	Usage   *chatUsage     `json:"usage,omitempty"`
}

type streamChoice struct {
	Index        int         `json:"index"`
	Delta        streamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type streamDelta struct {
	Role             string           `json:"role,omitempty"`
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []streamToolCall `json:"tool_calls,omitempty"`
}

type streamToolCall struct {
	Index    int             `json:"index"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Function *streamFuncCall `json:"function,omitempty"`
}

type streamFuncCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Chat 发送消息并获取完整响应
func (p *OpenAICompatibleProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
	reqBody := p.buildChatRequest(messages, tools, false)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logrus.Errorf("LLM API返回错误, status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("LLM API返回错误(status=%d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM返回空响应")
	}

	choice := chatResp.Choices[0]
	llmResp := &LLMResponse{
		Content:          choice.Message.Content,
		ReasoningContent: choice.Message.ReasoningContent,
		ToolCalls:        make([]ToolCall, 0),
		FinishReason:     choice.FinishReason,
		Usage: Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}

	// 转换工具调用
	for _, tc := range choice.Message.ToolCalls {
		llmResp.ToolCalls = append(llmResp.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return llmResp, nil
}

// ChatStream 流式发送消息
func (p *OpenAICompatibleProvider) ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	reqBody := p.buildChatRequest(messages, tools, true)

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		// logrus.Errorf("LLM Stream API返回错误, status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("LLM Stream API返回错误(status=%d): %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		p.parseSSEStream(resp.Body, ch)
	}()

	return ch, nil
}

// parseSSEStream 解析SSE流式响应
func (p *OpenAICompatibleProvider) parseSSEStream(reader io.Reader, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(reader)
	// 增大缓冲区，处理长响应
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// 用于累积增量工具调用
	var toolCallsAccum []streamToolCall

	for scanner.Scan() {
		line := scanner.Text()

		// SSE数据行以 "data: " 开头
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 流结束标记
		if data == "[DONE]" {
			ch <- StreamChunk{Done: true}
			return
		}

		var streamResp streamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			logrus.Warnf("解析SSE数据失败: %v, data=%s", err, data)
			continue
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]
		chunk := StreamChunk{
			Content: choice.Delta.Content,
		}

		// 捕获推理内容（思考模式）
		if choice.Delta.ReasoningContent != "" {
			chunk.ReasoningContent = choice.Delta.ReasoningContent
		}

		// 处理增量工具调用
		if len(choice.Delta.ToolCalls) > 0 {
			for _, stc := range choice.Delta.ToolCalls {
				// 扩展累积数组
				for len(toolCallsAccum) <= stc.Index {
					toolCallsAccum = append(toolCallsAccum, streamToolCall{})
				}

				// 累积ID和名称
				if stc.ID != "" {
					toolCallsAccum[stc.Index].ID = stc.ID
					toolCallsAccum[stc.Index].Type = "function"
				}
				if stc.Function != nil {
					if stc.Function.Name != "" {
						toolCallsAccum[stc.Index].Function = &streamFuncCall{}
						toolCallsAccum[stc.Index].Function.Name = stc.Function.Name
					}
					if stc.Function.Arguments != "" {
						if toolCallsAccum[stc.Index].Function == nil {
							toolCallsAccum[stc.Index].Function = &streamFuncCall{}
						}
						toolCallsAccum[stc.Index].Function.Arguments += stc.Function.Arguments
					}
				}
			}
		}

		// 处理结束原因
		if choice.FinishReason != nil {
			chunk.FinishReason = *choice.FinishReason

			// 流结束时，将累积的工具调用转换为完整ToolCall
			if *choice.FinishReason == "tool_calls" || *choice.FinishReason == "stop" {
				if len(toolCallsAccum) > 0 {
					chunk.ToolCalls = make([]ToolCall, 0, len(toolCallsAccum))
					for _, tc := range toolCallsAccum {
						if tc.Function != nil {
							chunk.ToolCalls = append(chunk.ToolCalls, ToolCall{
								ID:   tc.ID,
								Type: "function",
								Function: FunctionCall{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							})
						}
					}
				}
			}
		}

		if streamResp.Usage != nil {
			usage := Usage{
				PromptTokens:     streamResp.Usage.PromptTokens,
				CompletionTokens: streamResp.Usage.CompletionTokens,
				TotalTokens:      streamResp.Usage.TotalTokens,
			}
			chunk.Usage = &usage
		}

		ch <- chunk
	}

	if err := scanner.Err(); err != nil {
		logrus.Errorf("读取SSE流失败: %v", err)
		ch <- StreamChunk{Done: true}
	}
}

// buildChatRequest 构建聊天请求
func (p *OpenAICompatibleProvider) buildChatRequest(messages []Message, tools []ToolDefinition, stream bool) chatRequest {
	req := chatRequest{
		Model:    p.modelName,
		Messages: make([]chatMessage, 0, len(messages)),
		Stream:   stream,
	}

	// 转换消息
	for _, msg := range messages {
		cm := chatMessage{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
			ToolCallID:       msg.ToolCallID,
			Name:             msg.Name,
		}
		for _, tc := range msg.ToolCalls {
			cm.ToolCalls = append(cm.ToolCalls, chatToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: chatFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		req.Messages = append(req.Messages, cm)
	}

	// 转换工具定义
	if len(tools) > 0 {
		req.Tools = make([]chatTool, 0, len(tools))
		for _, t := range tools {
			req.Tools = append(req.Tools, chatTool{
				Type: "function",
				Function: chatFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			})
		}
	}

	return req
}

// setHeaders 设置请求头
func (p *OpenAICompatibleProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
}

// GetModelName 获取模型名称
func (p *OpenAICompatibleProvider) GetModelName() string {
	return p.modelName
}

// GetProviderType 获取提供商类型
func (p *OpenAICompatibleProvider) GetProviderType() ProviderType {
	return p.providerType
}
