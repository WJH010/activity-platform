package controller

import (
	"event-platform/internal/agent/dto"
	"event-platform/internal/agent/service"
	"event-platform/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AgentSkillController Skill管理控制器
type AgentSkillController struct {
	skillSvc service.SkillService
}

// NewAgentSkillController 创建Skill管理控制器
func NewAgentSkillController(skillSvc service.SkillService) *AgentSkillController {
	return &AgentSkillController{skillSvc: skillSvc}
}

// ListSkills 获取所有Skill
// GET /api/agent/skills
func (ctr *AgentSkillController) ListSkills(ctx *gin.Context) {
	skills, err := ctr.skillSvc.List(ctx.Request.Context())
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "获取Skill列表成功", skills)
}

// GetSkill 根据ID获取Skill
// GET /api/agent/skills/:id
func (ctr *AgentSkillController) GetSkill(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的Skill ID"))
		return
	}

	skill, err := ctr.skillSvc.GetByID(ctx.Request.Context(), id)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "获取Skill成功", skill)
}

// CreateSkill 创建动态Skill
// POST /api/agent/skills
func (ctr *AgentSkillController) CreateSkill(ctx *gin.Context) {
	var req dto.CreateSkillRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.skillSvc.Create(ctx.Request.Context(), req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "创建Skill成功", nil)
}

// UpdateSkill 更新Skill
// PUT /api/agent/skills/:id
func (ctr *AgentSkillController) UpdateSkill(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的Skill ID"))
		return
	}

	var req dto.UpdateSkillRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.skillSvc.Update(ctx.Request.Context(), id, req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "更新Skill成功", nil)
}

// DeleteSkill 删除动态Skill
// DELETE /api/agent/skills/:id
func (ctr *AgentSkillController) DeleteSkill(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的Skill ID"))
		return
	}

	if err := ctr.skillSvc.Delete(ctx.Request.Context(), id); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "删除Skill成功", nil)
}

// ToggleSkill 启用/禁用Skill
// PUT /api/agent/skills/:id/toggle
func (ctr *AgentSkillController) ToggleSkill(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeParamInvalid, "无效的Skill ID"))
		return
	}

	var req dto.ToggleSkillRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.skillSvc.Toggle(ctx.Request.Context(), id, req.IsEnabled); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "操作成功", nil)
}
