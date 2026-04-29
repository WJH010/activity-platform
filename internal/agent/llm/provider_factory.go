package llm

import (
	"fmt"
)

// ProviderConfig Provider配置
type ProviderConfig struct {
	ProviderType ProviderType
	BaseURL      string
	ApiKey       string
	ModelName    string
}

// NewProvider 根据配置创建LLM Provider
// 所有Provider统一走OpenAI兼容协议，仅baseURL和apiKey不同
func NewProvider(config ProviderConfig) (LLMProvider, error) {
	if err := validateProviderConfig(config); err != nil {
		return nil, err
	}

	baseURL := normalizeBaseURL(config.ProviderType, config.BaseURL)

	return NewOpenAICompatibleProvider(
		config.ProviderType,
		baseURL,
		config.ApiKey,
		config.ModelName,
	), nil
}

// normalizeBaseURL 规范化各Provider的baseURL
// 确保URL格式正确，补全路径
func normalizeBaseURL(providerType ProviderType, baseURL string) string {
	// 如果用户已经提供了完整路径，直接使用
	// 否则根据Provider类型补充默认路径
	switch providerType {
	case ProviderOllama:
		// Ollama默认: http://localhost:11434/v1
		if baseURL == "" {
			return "http://localhost:11434/v1"
		}
	case ProviderQianwen:
		// 千问DashScope OpenAI兼容: https://dashscope.aliyuncs.com/compatible-mode/v1
		if baseURL == "" {
			return "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
	case ProviderDeepSeek:
		// DeepSeek: https://api.deepseek.com/v1
		if baseURL == "" {
			return "https://api.deepseek.com/v1"
		}
	case ProviderDoubao:
		// 豆包(火山引擎Ark): https://ark.cn-beijing.volces.com/api/v3
		if baseURL == "" {
			return "https://ark.cn-beijing.volces.com/api/v3"
		}
	}
	return baseURL
}

// validateProviderConfig 验证Provider配置
func validateProviderConfig(config ProviderConfig) error {
	switch config.ProviderType {
	case ProviderOllama:
		// Ollama本地部署不需要apiKey
	case ProviderQianwen, ProviderDeepSeek, ProviderDoubao:
		// 云API需要apiKey
		if config.ApiKey == "" {
			return fmt.Errorf("Provider %s 需要配置API Key", config.ProviderType)
		}
	default:
		return fmt.Errorf("不支持的Provider类型: %s", config.ProviderType)
	}

	if config.ModelName == "" {
		return fmt.Errorf("模型名称不能为空")
	}

	return nil
}

// GetProviderDisplayInfo 获取Provider类型的信息（供前端展示）
func GetProviderDisplayInfo() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type":           string(ProviderOllama),
			"display_name":   "Ollama (本地模型)",
			"default_url":    "http://localhost:11434/v1",
			"need_api_key":   false,
			"popular_models": []string{"qwen2.5:7b", "llama3.1:8b", "deepseek-r1:7b"},
		},
		{
			"type":           string(ProviderQianwen),
			"display_name":   "通义千问",
			"default_url":    "https://dashscope.aliyuncs.com/compatible-mode/v1",
			"need_api_key":   true,
			"popular_models": []string{"qwen-plus", "qwen-turbo", "qwen-max"},
		},
		{
			"type":           string(ProviderDeepSeek),
			"display_name":   "DeepSeek",
			"default_url":    "https://api.deepseek.com/v1",
			"need_api_key":   true,
			"popular_models": []string{"deepseek-chat", "deepseek-reasoner"},
		},
		{
			"type":           string(ProviderDoubao),
			"display_name":   "豆包(火山引擎)",
			"default_url":    "https://ark.cn-beijing.volces.com/api/v3",
			"need_api_key":   true,
			"popular_models": []string{"doubao-pro-4k", "doubao-pro-32k", "doubao-pro-128k"},
		},
	}
}
