package dto

// CreateLLMConfigRequest 创建LLM配置请求
type CreateLLMConfigRequest struct {
	ProviderType string `json:"provider_type" binding:"required,oneof=ollama qianwen deepseek doubao"`
	DisplayName  string `json:"display_name" binding:"required,max=100"`
	ModelName    string `json:"model_name" binding:"required,max=100"`
	ApiUrl       string `json:"api_url" binding:"required,max=500"`
	ApiKey       string `json:"api_key" binding:"max=500"`
	ExtraConfig  string `json:"extra_config"`
}

// UpdateLLMConfigRequest 更新LLM配置请求
type UpdateLLMConfigRequest struct {
	ProviderType *string `json:"provider_type" binding:"omitempty,oneof=ollama qianwen deepseek doubao"`
	DisplayName  *string `json:"display_name" binding:"omitempty,non_empty_string,max=100"`
	ModelName    *string `json:"model_name" binding:"omitempty,non_empty_string,max=100"`
	ApiUrl       *string `json:"api_url" binding:"omitempty,non_empty_string,max=500"`
	ApiKey       *string `json:"api_key" binding:"omitempty,max=500"`
	ExtraConfig  *string `json:"extra_config"`
	IsEnabled    *int    `json:"is_enabled" binding:"omitempty,oneof=0 1"`
}

// LLMConfigResponse LLM配置响应
type LLMConfigResponse struct {
	ID           int    `json:"id"`
	ProviderType string `json:"provider_type"`
	DisplayName  string `json:"display_name"`
	ModelName    string `json:"model_name"`
	ApiUrl       string `json:"api_url"`
	ApiKey       string `json:"api_key,omitempty"` // 列表接口不返回完整key
	ExtraConfig  string `json:"extra_config"`
	IsEnabled    int    `json:"is_enabled"`
}

// SetUserLLMRequest 设置用户默认LLM请求
type SetUserLLMRequest struct {
	LLMConfigID int `json:"llm_config_id" binding:"required"`
}

// TestLLMConnectionRequest 测试LLM连接请求
type TestLLMConnectionRequest struct {
	ProviderType string `json:"provider_type" binding:"required,oneof=ollama qianwen deepseek doubao"`
	ApiUrl       string `json:"api_url" binding:"required,max=500"`
	ApiKey       string `json:"api_key" binding:"max=500"`
	ModelName    string `json:"model_name" binding:"required,max=100"`
}
