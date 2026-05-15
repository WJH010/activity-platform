package engine

import (
	"context"
	"encoding/json"
	"event-platform/internal/agent/llm"
	"event-platform/internal/agent/skill"
	"fmt"
	"strings"
	"unicode/utf8"
)

// EngineConfig 引擎配置
type EngineConfig struct {
	MaxReactRounds     int
	MaxHistoryMessages int
}

// Engine Agent引擎
type Engine struct {
	provider  llm.LLMProvider
	registry  *skill.Registry
	config    EngineConfig
	userID    int
	authToken string
}

// NewEngine 创建Agent引擎
func NewEngine(provider llm.LLMProvider, registry *skill.Registry, config EngineConfig, userID int, authToken string) *Engine {
	if config.MaxReactRounds <= 0 {
		config.MaxReactRounds = 10
	}
	if config.MaxHistoryMessages <= 0 {
		config.MaxHistoryMessages = 50
	}
	return &Engine{
		provider:  provider,
		registry:  registry,
		config:    config,
		userID:    userID,
		authToken: authToken,
	}
}

// RunStream 流式运行ReAct循环
func (e *Engine) RunStream(ctx context.Context, messages []llm.Message) (<-chan AgentEvent, error) {
	tools := e.registry.GetAllToolDefinitions()
	eventCh := make(chan AgentEvent, 200)

	go func() {
		defer close(eventCh)
		e.runReactLoop(ctx, messages, tools, eventCh)
	}()

	return eventCh, nil
}

// Run 非流式运行ReAct循环
func (e *Engine) Run(ctx context.Context, messages []llm.Message) (*AgentResult, error) {
	tools := e.registry.GetAllToolDefinitions()
	workingMessages := make([]llm.Message, len(messages))
	copy(workingMessages, messages)

	var finalContent strings.Builder
	var toolCallRecords []ToolCallRecord

	// 非流式ReAct循环
	for round := 0; round < e.config.MaxReactRounds; round++ {
		resp, err := e.provider.Chat(ctx, workingMessages, tools)
		if err != nil {
			return nil, fmt.Errorf("LLM调用失败: %w", err)
		}

		assistantMsg := llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		workingMessages = append(workingMessages, assistantMsg)
		finalContent.WriteString(resp.Content)

		if len(resp.ToolCalls) == 0 || resp.FinishReason == "stop" {
			break
		}

		// 处理工具调用
		for _, tc := range resp.ToolCalls {
			record := e.executeToolCall(ctx, tc)
			toolCallRecords = append(toolCallRecords, record)

			toolMsg := llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    record.ResultMsg,
			}
			workingMessages = append(workingMessages, toolMsg)
		}
	}

	return &AgentResult{
		Content:         finalContent.String(),
		ToolCallRecords: toolCallRecords,
	}, nil
}

// runReactLoop 流式ReAct循环
func (e *Engine) runReactLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, eventCh chan<- AgentEvent) {
	workingMessages := make([]llm.Message, len(messages))
	copy(workingMessages, messages)

	var totalReasoningContent string // 跨轮次累积推理内容

	// 流式ReAct循环
	for round := 0; round < e.config.MaxReactRounds; round++ {
		// logrus.Debugf("[AgentEngine] ReAct Round %d/%d", round+1, e.config.MaxReactRounds)

		streamCh, err := e.provider.ChatStream(ctx, workingMessages, tools)
		if err != nil {
			eventCh <- AgentEvent{
				Type:  EventError,
				Error: fmt.Sprintf("LLM调用失败: %s", err.Error()),
			}
			return
		}

		var contentBuf strings.Builder
		var reasoningBuf strings.Builder // 本轮推理内容累积
		var toolCalls []llm.ToolCall

		// 处理流式响应,逐个处理每个chunk，累积推理内容和工具调用
		for chunk := range streamCh {
			if chunk.Content != "" {
				contentBuf.WriteString(chunk.Content)
				eventCh <- AgentEvent{
					Type:    EventContent,
					Content: chunk.Content,
				}
			}

			// 累积推理内容（不发送给前端，仅用于回传给LLM和持久化）
			if chunk.ReasoningContent != "" {
				reasoningBuf.WriteString(chunk.ReasoningContent)
			}

			if len(chunk.ToolCalls) > 0 {
				toolCalls = chunk.ToolCalls
			}

			if chunk.Done {
				break
			}
		}

		// 累加本轮推理内容到总轮次推理内容
		roundReasoning := reasoningBuf.String()
		totalReasoningContent += roundReasoning

		assistantMsg := llm.Message{
			Role:             "assistant",
			Content:          contentBuf.String(),
			ReasoningContent: roundReasoning, // 关键：保留推理内容，回传给LLM
			ToolCalls:        toolCalls,
		}
		workingMessages = append(workingMessages, assistantMsg)

		// 没有工具调用，对话结束
		if len(toolCalls) == 0 {
			eventCh <- AgentEvent{
				Type:             EventDone,
				ReasoningContent: totalReasoningContent, // 携带推理内容供持久化
			}
			return
		}

		// 执行工具调用
		for _, tc := range toolCalls {
			eventCh <- AgentEvent{
				Type:       EventToolCall,
				ToolName:   tc.Function.Name,
				ToolArgs:   tc.Function.Arguments,
				ToolCallID: tc.ID,
			}

			//logrus.Infof("[AgentEngine] 执行工具: %s, 参数: %s", tc.Function.Name, tc.Function.Arguments)
			record := e.executeToolCall(ctx, tc)

			eventCh <- AgentEvent{
				Type:       EventToolResult,
				ToolName:   tc.Function.Name,
				ToolCallID: tc.ID,
				Result:     record.Result,
				Success:    record.Success,
			}

			toolMsg := llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    record.ResultMsg,
			}
			workingMessages = append(workingMessages, toolMsg)
		}
	}

	// logrus.Warnf("[AgentEngine] ReAct循环达到最大轮数 %d", e.config.MaxReactRounds)
	eventCh <- AgentEvent{
		Type:             EventDone,
		ReasoningContent: totalReasoningContent,
	}
}

// executeToolCall 执行单个工具调用
func (e *Engine) executeToolCall(ctx context.Context, tc llm.ToolCall) ToolCallRecord {
	record := ToolCallRecord{
		ToolCallID: tc.ID,
		ToolName:   tc.Function.Name,
		Arguments:  tc.Function.Arguments,
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
		// logrus.Errorf("[AgentEngine] 解析工具参数失败: %v, args=%s", err, tc.Function.Arguments)
		record.Success = false
		record.ResultMsg = fmt.Sprintf("参数解析失败: %s", err.Error())
		return record
	}

	skillCtx := skill.SkillContext{
		Ctx:       ctx,
		Params:    params,
		UserID:    e.userID,
		AuthToken: e.authToken,
	}

	result, err := e.registry.Execute(tc.Function.Name, skillCtx)
	if err != nil {
		// logrus.Errorf("[AgentEngine] 执行工具 %s 失败: %v", tc.Function.Name, err)
		record.Success = false
		record.ResultMsg = fmt.Sprintf("工具执行失败: %s", err.Error())
		return record
	}

	record.Success = result.Success
	record.Result = result.Data
	record.ResultMsg = e.buildToolResultMessage(result)

	return record
}

// buildToolResultMessage 构建给LLM的工具结果消息
// 增加8K字符硬截断，防止上下文溢出
const maxToolResultChars = 8000

func (e *Engine) buildToolResultMessage(result skill.SkillResult) string {
	msg := result.Message

	if result.Data != nil {
		dataBytes, err := json.Marshal(result.Data)
		if err == nil {
			var combined string
			if msg != "" {
				combined = msg + "\n" + string(dataBytes)
			} else {
				combined = string(dataBytes)
			}
			return truncateWithHint(combined, maxToolResultChars)
		}
	}

	if msg == "" {
		if result.Success {
			return "操作成功"
		}
		return "操作失败"
	}

	return truncateWithHint(msg, maxToolResultChars)
}

// truncateWithHint 截断字符串，超出时添加提示
func truncateWithHint(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen]) + "\n...[内容过长已截断，如需完整数据请使用详情查询工具]"
}
