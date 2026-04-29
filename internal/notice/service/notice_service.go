package service

import (
	"activity-platform/internal/notice/model"
	"activity-platform/internal/notice/repository"
	"context"
)

// NoticeService 服务接口，定义方法，接收 context.Context 和数据模型。
type NoticeService interface {
	ListNotice(ctx context.Context, page, pageSize int) ([]*model.Notice, int64, error)
	GetNoticeContent(ctx context.Context, noticeID int) (*model.Notice, error)
}

// NoticeServiceImpl 实现接口的具体结构体，持有数据访问层接口 Repository 的实例
type NoticeServiceImpl struct {
	noticeRepo repository.NoticeRepository
}

// NewNoticeService 创建服务实例
func NewNoticeService(noticeRepo repository.NoticeRepository) NoticeService {
	return &NoticeServiceImpl{noticeRepo: noticeRepo}
}

// ListNotice 分页查询数据
func (svc *NoticeServiceImpl) ListNotice(ctx context.Context, page, pageSize int) ([]*model.Notice, int64, error) {
	return svc.noticeRepo.List(ctx, page, pageSize)
}

// GetNoticeContent 查询公告内容
func (svc *NoticeServiceImpl) GetNoticeContent(ctx context.Context, noticeID int) (*model.Notice, error) {
	return svc.noticeRepo.GetNoticeContent(ctx, noticeID)
}
