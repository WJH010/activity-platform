package service

import (
	"activity-platform/internal/agent/dto"
	"activity-platform/internal/agent/model"
	"activity-platform/internal/agent/repository"
	"activity-platform/internal/utils"
	"context"
	"strings"
)

// LLMConfigService LLM配置管理服务接口
type LLMConfigService interface {
	// List 获取所有LLM配置
	List(ctx context.Context) ([]dto.LLMConfigResponse, error)
	// GetByID 根据ID获取LLM配置
	GetByID(ctx context.Context, id int) (*dto.LLMConfigResponse, error)
	// Create 创建LLM配置
	Create(ctx context.Context, req dto.CreateLLMConfigRequest) error
	// Update 更新LLM配置
	Update(ctx context.Context, id int, req dto.UpdateLLMConfigRequest) error
	// Delete 删除LLM配置
	Delete(ctx context.Context, id int) error
	// SetUserLLM 设置用户默认LLM
	SetUserLLM(ctx context.Context, userID int, req dto.SetUserLLMRequest) error
	// GetUserLLMConfig 获取用户选择的LLM配置
	GetUserLLMConfig(ctx context.Context, userID int) (*model.LLMConfig, error)
}

type llmConfigServiceImpl struct {
	llmConfigRepo   repository.LLMConfigRepository
	userLLMPrefRepo repository.UserLLMPrefRepository
}

func NewLLMConfigService(
	llmConfigRepo repository.LLMConfigRepository,
	userLLMPrefRepo repository.UserLLMPrefRepository,
) LLMConfigService {
	return &llmConfigServiceImpl{
		llmConfigRepo:   llmConfigRepo,
		userLLMPrefRepo: userLLMPrefRepo,
	}
}

func (svc *llmConfigServiceImpl) List(ctx context.Context) ([]dto.LLMConfigResponse, error) {
	configs, err := svc.llmConfigRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.LLMConfigResponse, 0, len(configs))
	for _, c := range configs {
		result = append(result, dto.LLMConfigResponse{
			ID:           c.ID,
			ProviderType: c.ProviderType,
			DisplayName:  c.DisplayName,
			ModelName:    c.ModelName,
			ApiUrl:       c.ApiUrl,
			ApiKey:       maskApiKey(c.ApiKey),
			ExtraConfig:  c.ExtraConfig,
			IsEnabled:    c.IsEnabled,
		})
	}
	return result, nil
}

func (svc *llmConfigServiceImpl) GetByID(ctx context.Context, id int) (*dto.LLMConfigResponse, error) {
	config, err := svc.llmConfigRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &dto.LLMConfigResponse{
		ID:           config.ID,
		ProviderType: config.ProviderType,
		DisplayName:  config.DisplayName,
		ModelName:    config.ModelName,
		ApiUrl:       config.ApiUrl,
		ApiKey:       maskApiKey(config.ApiKey),
		ExtraConfig:  config.ExtraConfig,
		IsEnabled:    config.IsEnabled,
	}, nil
}

func (svc *llmConfigServiceImpl) Create(ctx context.Context, req dto.CreateLLMConfigRequest) error {
	config := &model.LLMConfig{
		ProviderType: req.ProviderType,
		DisplayName:  req.DisplayName,
		ModelName:    req.ModelName,
		ApiUrl:       req.ApiUrl,
		ApiKey:       req.ApiKey,
		ExtraConfig:  req.ExtraConfig,
		IsEnabled:    1,
	}
	return svc.llmConfigRepo.Create(ctx, config)
}

func (svc *llmConfigServiceImpl) Update(ctx context.Context, id int, req dto.UpdateLLMConfigRequest) error {
	updateFields := make(map[string]interface{})
	if req.ProviderType != nil {
		updateFields["provider_type"] = *req.ProviderType
	}
	if req.DisplayName != nil {
		updateFields["display_name"] = *req.DisplayName
	}
	if req.ModelName != nil {
		updateFields["model_name"] = *req.ModelName
	}
	if req.ApiUrl != nil {
		updateFields["api_url"] = *req.ApiUrl
	}
	if req.ApiKey != nil {
		updateFields["api_key"] = *req.ApiKey
	}
	if req.ExtraConfig != nil {
		updateFields["extra_config"] = *req.ExtraConfig
	}
	if req.IsEnabled != nil {
		updateFields["is_enabled"] = *req.IsEnabled
	}
	if len(updateFields) == 0 {
		return nil
	}
	return svc.llmConfigRepo.Update(ctx, id, updateFields)
}

func (svc *llmConfigServiceImpl) Delete(ctx context.Context, id int) error {
	return svc.llmConfigRepo.Delete(ctx, id)
}

func (svc *llmConfigServiceImpl) SetUserLLM(ctx context.Context, userID int, req dto.SetUserLLMRequest) error {
	// 检查配置是否存在
	config, err := svc.llmConfigRepo.GetByID(ctx, req.LLMConfigID)
	if err != nil {
		return err
	}
	if config.IsEnabled != 1 {
		return utils.NewBusinessError(utils.ErrCodeBusinessLogicError, "该LLM配置未启用")
	}

	pref := &model.UserLLMPref{
		UserID:      userID,
		LLMConfigID: req.LLMConfigID,
	}
	return svc.userLLMPrefRepo.Upsert(ctx, pref)
}

func (svc *llmConfigServiceImpl) GetUserLLMConfig(ctx context.Context, userID int) (*model.LLMConfig, error) {
	pref, err := svc.userLLMPrefRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if pref == nil {
		return nil, nil
	}
	return svc.llmConfigRepo.GetByID(ctx, pref.LLMConfigID)
}

// maskApiKey 脱敏API Key
func maskApiKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
