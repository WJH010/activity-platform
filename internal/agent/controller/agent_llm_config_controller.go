package controller

import (
	"activity-platform/internal/agent/dto"
	"activity-platform/internal/agent/llm"
	"activity-platform/internal/agent/service"
	"activity-platform/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AgentLLMConfigController LLM配置管理控制器
type AgentLLMConfigController struct {
	llmConfigSvc service.LLMConfigService
}

// NewAgentLLMConfigController 创建LLM配置管理控制器
func NewAgentLLMConfigController(llmConfigSvc service.LLMConfigService) *AgentLLMConfigController {
	return &AgentLLMConfigController{llmConfigSvc: llmConfigSvc}
}

// ListConfigs 获取所有LLM配置
// GET /api/agent/llm-configs
func (ctr *AgentLLMConfigController) ListConfigs(ctx *gin.Context) {
	configs, err := ctr.llmConfigSvc.List(ctx.Request.Context())
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "获取LLM配置列表成功", configs)
}

// GetConfig 根据ID获取LLM配置
// GET /api/agent/llm-configs/:id
func (ctr *AgentLLMConfigController) GetConfig(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的配置ID"))
		return
	}

	config, err := ctr.llmConfigSvc.GetByID(ctx.Request.Context(), id)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "获取LLM配置成功", config)
}

// CreateConfig 创建LLM配置
// POST /api/agent/llm-configs
func (ctr *AgentLLMConfigController) CreateConfig(ctx *gin.Context) {
	var req dto.CreateLLMConfigRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.llmConfigSvc.Create(ctx.Request.Context(), req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "创建LLM配置成功", nil)
}

// UpdateConfig 更新LLM配置
// PUT /api/agent/llm-configs/:id
func (ctr *AgentLLMConfigController) UpdateConfig(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的配置ID"))
		return
	}

	var req dto.UpdateLLMConfigRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.llmConfigSvc.Update(ctx.Request.Context(), id, req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "更新LLM配置成功", nil)
}

// DeleteConfig 删除LLM配置
// DELETE /api/agent/llm-configs/:id
func (ctr *AgentLLMConfigController) DeleteConfig(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的配置ID"))
		return
	}

	if err := ctr.llmConfigSvc.Delete(ctx.Request.Context(), id); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "删除LLM配置成功", nil)
}

// SetUserLLM 设置用户默认LLM
// POST /api/agent/llm-configs/user-preference
func (ctr *AgentLLMConfigController) SetUserLLM(ctx *gin.Context) {
	userID, err := utils.GetUserID(ctx)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	var req dto.SetUserLLMRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.llmConfigSvc.SetUserLLM(ctx.Request.Context(), userID, req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "设置成功", nil)
}

// GetProviderInfo 获取Provider类型信息（供前端展示和选择）
// GET /api/agent/llm-configs/providers
func (ctr *AgentLLMConfigController) GetProviderInfo(ctx *gin.Context) {
	info := llm.GetProviderDisplayInfo()
	utils.Success(ctx, "获取Provider信息成功", info)
}
