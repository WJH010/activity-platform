package service

import (
	"activity-platform/internal/user/dto"
	"activity-platform/internal/user/model"
	"activity-platform/internal/user/repository"
	"activity-platform/internal/utils"
	"context"
)

// IndustryService 服务接口，定义行业相关的业务逻辑方法
type IndustryService interface {
	// ListIndustries 查询行业列表
	ListIndustries(ctx context.Context) ([]*model.Industries, error)
	// CreateIndustry 创建行业
	CreateIndustry(ctx context.Context, industry *model.Industries) error
	// UpdateIndustry 更新行业信息
	UpdateIndustry(ctx context.Context, industryID int, req dto.UpdateIndustryRequest) error
	// DeleteIndustry 删除行业
	DeleteIndustry(ctx context.Context, industryID int) error
}

// IndustryServiceImpl 实现 IndustryService 接口，提供行业相关的业务逻辑
type IndustryServiceImpl struct {
	industryRepo repository.IndustryRepository // 行业数据访问接口
}

// NewIndustryService 创建服务实例
func NewIndustryService(industryRepo repository.IndustryRepository) IndustryService {
	return &IndustryServiceImpl{
		industryRepo: industryRepo,
	}
}

// ListIndustries 查询行业列表
func (svc *IndustryServiceImpl) ListIndustries(ctx context.Context) ([]*model.Industries, error) {
	// 调用数据访问层的查询方法
	return svc.industryRepo.ListIndustries(ctx)
}

// CreateIndustry 创建行业
func (svc *IndustryServiceImpl) CreateIndustry(ctx context.Context, industry *model.Industries) error {
	// 调用数据访问层的创建方法
	return svc.industryRepo.CreateIndustry(ctx, industry)
}

// UpdateIndustry 更新行业信息
func (svc *IndustryServiceImpl) UpdateIndustry(ctx context.Context, industryID int, req dto.UpdateIndustryRequest) error {
	// 构建更新字段
	updateFields := make(map[string]interface{})
	if req.IndustryCode != "" {
		updateFields["industry_code"] = req.IndustryCode
	}
	if req.IndustryName != "" {
		updateFields["industry_name"] = req.IndustryName
	}
	// 如果没有更新字段，则直接返回
	if len(updateFields) == 0 {
		return nil
	}

	// 调用数据访问层的更新方法
	return svc.industryRepo.UpdateIndustry(ctx, industryID, updateFields)
}

// DeleteIndustry 删除行业
func (svc *IndustryServiceImpl) DeleteIndustry(ctx context.Context, industryID int) error {
	// 执行软删除，直接调用更新过程
	updateFields := make(map[string]interface{})
	updateFields["is_deleted"] = utils.DeletedFlagYes

	return svc.industryRepo.UpdateIndustry(ctx, industryID, updateFields)
}
