package service

import (
	"activity-platform/internal/agent/dto"
	"activity-platform/internal/agent/engine"
	"activity-platform/internal/agent/llm"
	"activity-platform/internal/agent/model"
	"activity-platform/internal/agent/repository"
	"activity-platform/internal/agent/skill"
	"activity-platform/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// AgentChatService Agent对话服务接口
type AgentChatService interface {
	// ChatStream 流式对话，返回事件channel和会话ID
	ChatStream(ctx context.Context, userID int, authToken string, req dto.ChatRequest) (<-chan engine.AgentEvent, string, error)
	// LoadSessionMessages 加载会话消息历史
	LoadSessionMessages(ctx context.Context, sessionID string, userID int) ([]dto.MessageResponse, error)
}

type agentChatServiceImpl struct {
	sessionSvc    SessionService
	llmConfigSvc  LLMConfigService
	llmConfigRepo repository.LLMConfigRepository
	messageRepo   repository.MessageRepository
	skillSvc      SkillService
	skillReg      *skill.Registry
	cfg           *config.Config
}

func NewAgentChatService(
	sessionSvc SessionService,
	llmConfigSvc LLMConfigService,
	llmConfigRepo repository.LLMConfigRepository,
	messageRepo repository.MessageRepository,
	skillSvc SkillService,
	skillReg *skill.Registry,
	cfg *config.Config,
) AgentChatService {
	return &agentChatServiceImpl{
		sessionSvc:    sessionSvc,
		llmConfigSvc:  llmConfigSvc,
		llmConfigRepo: llmConfigRepo,
		messageRepo:   messageRepo,
		skillSvc:      skillSvc,
		skillReg:      skillReg,
		cfg:           cfg,
	}
}

// ChatStream 流式对话
func (svc *agentChatServiceImpl) ChatStream(ctx context.Context, userID int, authToken string, req dto.ChatRequest) (<-chan engine.AgentEvent, string, error) {
	// 1. 获取或创建会话
	session, err := svc.sessionSvc.GetOrCreateSession(ctx, userID, req.SessionID, nil)
	if err != nil {
		return nil, "", fmt.Errorf("获取会话失败: %w", err)
	}
	sessionID := session.ID

	// 2. 解析用户LLM配置 → 创建Provider
	provider, err := svc.createProvider(ctx, userID, session.LLMConfigID)
	if err != nil {
		return nil, sessionID, fmt.Errorf("创建LLM Provider失败: %w", err)
	}

	// 3. 加载历史消息
	historyMessages, err := svc.loadHistoryMessages(ctx, sessionID)
	if err != nil {
		logrus.Warnf("加载历史消息失败: %v, 将继续无历史消息对话", err)
		historyMessages = []llm.Message{}
	}

	// 4. 构建系统提示词
	systemPrompt := engine.BuildSystemPrompt(userID, "")

	// 5. 构建完整消息列表
	messages := make([]llm.Message, 0, len(historyMessages)+2)
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	messages = append(messages, historyMessages...)
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: req.Content,
	})

	// 6. 创建引擎
	engineConfig := engine.EngineConfig{
		MaxReactRounds:     svc.cfg.Agent.MaxReactRounds,
		MaxHistoryMessages: svc.cfg.Agent.MaxHistoryMessages,
	}
	eng := engine.NewEngine(provider, svc.skillReg, engineConfig, userID, authToken)

	// 7. 运行引擎（流式）
	engineEventCh, err := eng.RunStream(ctx, messages)
	if err != nil {
		return nil, sessionID, fmt.Errorf("启动Agent引擎失败: %w", err)
	}

	// 8. 创建SSE专用channel（不再和持久化共享同一个channel）
	sseCh := make(chan engine.AgentEvent, 200)

	// 9. 启动fan-out goroutine：独占消费engineEventCh，同时完成SSE转发和持久化
	needGenerateTitle := session.Title == ""
	go svc.fanOutAndPersist(engineEventCh, sseCh, sessionID, userID, req.Content, provider, needGenerateTitle)

	return sseCh, sessionID, nil
}

// LoadSessionMessages 加载会话消息
func (svc *agentChatServiceImpl) LoadSessionMessages(ctx context.Context, sessionID string, userID int) ([]dto.MessageResponse, error) {
	return svc.sessionSvc.GetSessionMessages(ctx, sessionID, userID)
}

// createProvider 根据用户配置创建LLM Provider
func (svc *agentChatServiceImpl) createProvider(ctx context.Context, userID int, llmConfigID *int) (llm.LLMProvider, error) {
	var llmConfig *model.LLMConfig
	var err error

	// 优先使用会话指定的LLM配置
	if llmConfigID != nil {
		llmConfig, err = svc.llmConfigRepo.GetByID(ctx, *llmConfigID)
		if err != nil {
			logrus.Warnf("获取会话LLM配置失败: %v, 将回退到用户偏好", err)
			llmConfig = nil
		} else if llmConfig.IsEnabled != 1 {
			llmConfig = nil
		}
	}

	// 回退到用户偏好
	if llmConfig == nil && userID > 0 {
		llmConfig, err = svc.llmConfigSvc.GetUserLLMConfig(ctx, userID)
		if err != nil {
			logrus.Warnf("获取用户LLM偏好失败: %v", err)
			llmConfig = nil
		}
	}

	// 回退到默认Provider
	if llmConfig == nil {
		provider, defaultErr := svc.createDefaultProvider()
		if defaultErr != nil {
			logrus.Errorf("所有LLM配置回退均失败: %v", defaultErr)
			return nil, fmt.Errorf("无法创建AI服务，请在设置中配置有效的LLM服务（API Key和模型名称）")
		}
		return provider, nil
	}

	return llm.NewProvider(llm.ProviderConfig{
		ProviderType: llm.ProviderType(llmConfig.ProviderType),
		BaseURL:      llmConfig.ApiUrl,
		ApiKey:       llmConfig.ApiKey,
		ModelName:    llmConfig.ModelName,
	})
}

// createDefaultProvider 创建默认Provider
func (svc *agentChatServiceImpl) createDefaultProvider() (llm.LLMProvider, error) {
	defaultProvider := svc.cfg.Agent.DefaultProvider
	if defaultProvider == "" {
		defaultProvider = "deepseek"
	}
	provider, err := llm.NewProvider(llm.ProviderConfig{
		ProviderType: llm.ProviderType(defaultProvider),
		BaseURL:      "",
		ApiKey:       "",
		ModelName:    "",
	})
	if err != nil {
		return nil, fmt.Errorf("默认LLM Provider(%s)配置无效: %w", defaultProvider, err)
	}
	return provider, nil
}

// loadHistoryMessages 加载历史消息并转换为LLM消息格式
func (svc *agentChatServiceImpl) loadHistoryMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	messages, err := svc.messageRepo.ListBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	maxHistory := svc.cfg.Agent.MaxHistoryMessages
	if maxHistory <= 0 {
		maxHistory = 50
	}
	startIdx := 0
	if len(messages) > maxHistory {
		startIdx = len(messages) - maxHistory
	}

	result := make([]llm.Message, 0, len(messages)-startIdx)
	for i := startIdx; i < len(messages); i++ {
		msg := messages[i]
		llmMsg := llm.Message{
			Role:             msg.Role,
			Content:          msg.Content,
			ReasoningContent: msg.ReasoningContent,
		}

		if msg.ToolCalls != "" && msg.Role == "assistant" {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal([]byte(msg.ToolCalls), &toolCalls); err == nil {
				llmMsg.ToolCalls = toolCalls
			}
		}

		if msg.Role == "tool" {
			llmMsg.ToolCallID = msg.ToolCallID
			llmMsg.Name = msg.SkillName
			if msg.SkillResult != "" {
				llmMsg.Content = msg.SkillResult
			}
		}

		result = append(result, llmMsg)
	}

	return result, nil
}

// fanOutAndPersist 独占消费引擎事件channel，完成SSE转发和消息持久化
// 这是唯一的eventCh消费者，解决了channel竞争和context取消两个问题
func (svc *agentChatServiceImpl) fanOutAndPersist(
	engineEventCh <-chan engine.AgentEvent,
	sseCh chan<- engine.AgentEvent,
	sessionID string,
	userID int,
	userContent string,
	provider llm.LLMProvider,
	needGenerateTitle bool,
) {
	defer close(sseCh)

	// 使用独立的context，避免HTTP请求结束后context被取消
	persistCtx := context.Background()

	// 先保存用户消息
	userMsg := &model.Message{
		SessionID:  sessionID,
		Role:       "user",
		Content:    userContent,
		CreateUser: userID,
		UpdateUser: userID,
	}
	if err := svc.messageRepo.Create(persistCtx, userMsg); err != nil {
		logrus.Errorf("保存用户消息失败: %v", err)
	}

	// 累积assistant消息和工具调用
	var assistantContent string
	var toolCalls []llm.ToolCall
	var toolMessages []model.Message

	for event := range engineEventCh {
		// 转发所有事件到SSE channel（Controller从这里读取）
		sseCh <- event

		// 同时收集持久化所需的数据
		switch event.Type {
		case engine.EventContent:
			assistantContent += event.Content

		case engine.EventToolCall:
			toolCalls = append(toolCalls, llm.ToolCall{
				ID:   event.ToolCallID,
				Type: "function",
				Function: llm.FunctionCall{
					Name:      event.ToolName,
					Arguments: event.ToolArgs,
				},
			})

		case engine.EventToolResult:
			resultJSON := ""
			if event.Result != nil {
				if bytes, err := json.Marshal(event.Result); err == nil {
					resultJSON = string(bytes)
				}
			}

			statusMsg := "成功"
			if !event.Success {
				statusMsg = "失败"
			}

			toolMsg := model.Message{
				SessionID:   sessionID,
				Role:        "tool",
				ToolCallID:  event.ToolCallID,
				SkillName:   event.ToolName,
				SkillResult: resultJSON,
				Content:     fmt.Sprintf("工具 %s 执行%s", event.ToolName, statusMsg),
				CreateUser:  userID,
				UpdateUser:  userID,
			}
			toolMessages = append(toolMessages, toolMsg)

		case engine.EventDone:
			// 保存assistant消息
			var toolCallsJSON string
			if len(toolCalls) > 0 {
				if bytes, err := json.Marshal(toolCalls); err == nil {
					toolCallsJSON = string(bytes)
				}
			}

			assistantMsg := &model.Message{
				SessionID:        sessionID,
				Role:             "assistant",
				Content:          assistantContent,
				ReasoningContent: event.ReasoningContent,
				ToolCalls:        toolCallsJSON,
				CreateUser:       userID,
				UpdateUser:       userID,
			}
			if err := svc.messageRepo.Create(persistCtx, assistantMsg); err != nil {
				logrus.Errorf("保存助手消息失败: %v", err)
			}

			// 批量保存tool消息
			if len(toolMessages) > 0 {
				if err := svc.messageRepo.BatchCreate(persistCtx, toolMessages); err != nil {
					logrus.Errorf("保存工具消息失败: %v", err)
				}
			}

			// 首次对话时生成会话标题
			if needGenerateTitle {
				logrus.Infof("[标题生成] 开始为会话 %s 生成标题, userContent=%q", sessionID, userContent)
				svc.generateAndSetTitle(persistCtx, sessionID, userContent, provider)
			}

		case engine.EventError:
			logrus.Errorf("Agent引擎错误: %s", event.Error)
		}
	}
}

// generateAndSetTitle 生成并设置会话标题（仅新会话首次对话时调用）
func (svc *agentChatServiceImpl) generateAndSetTitle(ctx context.Context, sessionID, userContent string, provider llm.LLMProvider) {
	title := svc.generateTitleByLLM(ctx, userContent, provider)
	if title == "" {
		title = fallbackTitle(userContent)
	}
	if err := svc.sessionSvc.UpdateSessionTitle(ctx, sessionID, title); err != nil {
		logrus.Warnf("更新会话标题失败: %v", err)
	}
}

// generateTitleByLLM 调用LLM根据用户消息生成10字以内的标题
func (svc *agentChatServiceImpl) generateTitleByLLM(ctx context.Context, userContent string, provider llm.LLMProvider) string {
	prompt := fmt.Sprintf(
		"请根据以下用户消息，生成一个10字以内的简短标题，直接输出标题内容，不要加引号、不要加任何解释：\n\n%s",
		userContent,
	)

	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	resp, err := provider.Chat(ctx, messages, nil)
	if err != nil {
		logrus.Warnf("LLM生成标题失败，将使用fallback: %v", err)
		return ""
	}

	title := strings.TrimSpace(resp.Content)
	title = strings.Trim(title, `"'""'《》【】`)
	runes := []rune(title)
	if len(runes) > 10 {
		title = string(runes[:10])
	}
	if title == "" {
		return ""
	}
	return title
}

// fallbackTitle 当LLM生成失败时，截断用户消息作为标题
func fallbackTitle(userContent string) string {
	if userContent == "" {
		return "新对话"
	}
	runes := []rune(userContent)
	if len(runes) > 20 {
		return string(runes[:20]) + "..."
	}
	return userContent
}
