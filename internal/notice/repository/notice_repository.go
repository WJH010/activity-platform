package repository

import (
	"activity-platform/internal/notice/model"
	"activity-platform/internal/utils"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// NoticeRepository 数据访问接口，定义数据访问的方法集
type NoticeRepository interface {
	// List 分页查询公告列表
	List(ctx context.Context, page, pageSize int) ([]*model.Notice, int64, error)
	// GetNoticeContent 根据id获取公告内容
	GetNoticeContent(ctx context.Context, noticeID int) (*model.Notice, error)
}

// NoticeRepositoryImpl 实现接口的具体结构体
type NoticeRepositoryImpl struct {
	db *gorm.DB
}

// NewNoticeRepository 创建数据访问实例
func NewNoticeRepository(db *gorm.DB) NoticeRepository {
	return &NoticeRepositoryImpl{db: db}
}

// List 分页查询数据
func (repo *NoticeRepositoryImpl) List(ctx context.Context, page, pageSize int) ([]*model.Notice, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var notices []*model.Notice
	query := repo.db.WithContext(ctx)

	// 添加条件查询
	// 只查询未删除的公告
	query = query.Where("is_deleted = ?", utils.DeletedFlagNo)

	// 按发布时间降序排列
	query = query.Order("release_time DESC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Model(&model.Notice{}).Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&notices).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return notices, total, nil
}

// GetNoticeContent 内容查询
func (repo *NoticeRepositoryImpl) GetNoticeContent(ctx context.Context, noticeID int) (*model.Notice, error) {
	var notice model.Notice

	result := repo.db.WithContext(ctx).First(&notice, noticeID)
	err := result.Error

	// 查询公告内容
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "公告不存在或已被删除，请刷新页面后重试")
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &notice, nil
}
