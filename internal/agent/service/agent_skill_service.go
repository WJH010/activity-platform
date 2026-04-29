package service

import (
	"activity-platform/internal/agent/dto"
	"activity-platform/internal/agent/model"
	"activity-platform/internal/agent/repository"
	"activity-platform/internal/utils"
	"context"
)

// SkillService Skill管理服务接口
type SkillService interface {
	// List 获取所有Skill
	List(ctx context.Context) ([]dto.SkillResponse, error)
	// GetByID 根据ID获取Skill
	GetByID(ctx context.Context, id int) (*dto.SkillResponse, error)
	// Create 创建动态Skill
	Create(ctx context.Context, req dto.CreateSkillRequest) error
	// Update 更新Skill
	Update(ctx context.Context, id int, req dto.UpdateSkillRequest) error
	// Delete 删除动态Skill（内置Skill不可删除）
	Delete(ctx context.Context, id int) error
	// Toggle 启用/禁用Skill
	Toggle(ctx context.Context, id int, isEnabled int) error
	// ListEnabled 获取所有启用的Skill定义（供Agent引擎使用）
	ListEnabled(ctx context.Context) ([]model.SkillDefinition, error)
}

type skillServiceImpl struct {
	skillRepo repository.SkillRepository
}

func NewSkillService(skillRepo repository.SkillRepository) SkillService {
	return &skillServiceImpl{skillRepo: skillRepo}
}

func (svc *skillServiceImpl) List(ctx context.Context) ([]dto.SkillResponse, error) {
	skills, err := svc.skillRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]dto.SkillResponse, 0, len(skills))
	for _, s := range skills {
		result = append(result, toSkillResponse(&s))
	}
	return result, nil
}

func (svc *skillServiceImpl) GetByID(ctx context.Context, id int) (*dto.SkillResponse, error) {
	skill, err := svc.skillRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toSkillResponse(skill)
	return &resp, nil
}

func (svc *skillServiceImpl) Create(ctx context.Context, req dto.CreateSkillRequest) error {
	// 检查名称是否重复
	existing, err := svc.skillRepo.GetByName(ctx, req.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return utils.NewBusinessError(utils.ErrCodeResourceExists, "已存在同名Skill")
	}

	authRequired := 1
	if req.AuthRequired != nil {
		authRequired = *req.AuthRequired
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	skill := &model.SkillDefinition{
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		ParamSchema:     req.ParamSchema,
		HttpMethod:      req.HttpMethod,
		UrlTemplate:     req.UrlTemplate,
		HeadersTemplate: req.HeadersTemplate,
		BodyTemplate:    req.BodyTemplate,
		AuthRequired:    authRequired,
		IsEnabled:       1,
		IsBuiltin:       0,
		Category:        req.Category,
		SortOrder:       sortOrder,
	}
	return svc.skillRepo.Create(ctx, skill)
}

func (svc *skillServiceImpl) Update(ctx context.Context, id int, req dto.UpdateSkillRequest) error {
	// 检查Skill是否存在
	skill, err := svc.skillRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{})
	if req.DisplayName != nil {
		updateFields["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		updateFields["description"] = *req.Description
	}
	if req.ParamSchema != nil {
		updateFields["param_schema"] = *req.ParamSchema
	}
	if req.HttpMethod != nil {
		updateFields["http_method"] = *req.HttpMethod
	}
	if req.UrlTemplate != nil {
		updateFields["url_template"] = *req.UrlTemplate
	}
	if req.HeadersTemplate != nil {
		updateFields["headers_template"] = *req.HeadersTemplate
	}
	if req.BodyTemplate != nil {
		updateFields["body_template"] = *req.BodyTemplate
	}
	if req.AuthRequired != nil {
		updateFields["auth_required"] = *req.AuthRequired
	}
	if req.IsEnabled != nil {
		updateFields["is_enabled"] = *req.IsEnabled
	}
	if req.Category != nil {
		updateFields["category"] = *req.Category
	}
	if req.SortOrder != nil {
		updateFields["sort_order"] = *req.SortOrder
	}
	if len(updateFields) == 0 {
		return nil
	}

	_ = skill // 已确认存在
	return svc.skillRepo.Update(ctx, id, updateFields)
}

func (svc *skillServiceImpl) Delete(ctx context.Context, id int) error {
	skill, err := svc.skillRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if skill.IsBuiltin == 1 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotAllowed, "内置Skill不可删除")
	}
	return svc.skillRepo.Delete(ctx, id)
}

func (svc *skillServiceImpl) Toggle(ctx context.Context, id int, isEnabled int) error {
	return svc.skillRepo.Update(ctx, id, map[string]interface{}{
		"is_enabled": isEnabled,
	})
}

func (svc *skillServiceImpl) ListEnabled(ctx context.Context) ([]model.SkillDefinition, error) {
	return svc.skillRepo.ListEnabled(ctx)
}

func toSkillResponse(s *model.SkillDefinition) dto.SkillResponse {
	return dto.SkillResponse{
		ID:              s.ID,
		Name:            s.Name,
		DisplayName:     s.DisplayName,
		Description:     s.Description,
		ParamSchema:     s.ParamSchema,
		HttpMethod:      s.HttpMethod,
		UrlTemplate:     s.UrlTemplate,
		HeadersTemplate: s.HeadersTemplate,
		BodyTemplate:    s.BodyTemplate,
		AuthRequired:    s.AuthRequired,
		IsEnabled:       s.IsEnabled,
		IsBuiltin:       s.IsBuiltin,
		Category:        s.Category,
		SortOrder:       s.SortOrder,
	}
}
