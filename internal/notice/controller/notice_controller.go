package controller

import (
	"event-platform/internal/notice/dto"
	"event-platform/internal/notice/service"
	"event-platform/internal/utils"

	"github.com/gin-gonic/gin"
)

// NoticeController 控制器
type NoticeController struct {
	noticeService service.NoticeService
}

// NewNoticeController 创建控制器实例
func NewNoticeController(noticeService service.NoticeService) *NoticeController {
	return &NoticeController{noticeService: noticeService}
}

// ListNotice 分页查询公告列表
func (ctr *NoticeController) ListNotice(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.NoticeListRequest
	if !utils.BindQuery(ctx, &req) {
		return
	}

	// page 默认1
	page := req.Page
	if page == 0 {
		page = 1
	}

	// pageSize 默认10
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// 调用服务层
	notice, total, err := ctr.noticeService.ListNotice(ctx, page, pageSize)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	var result []dto.NoticeResponse
	for _, n := range notice {
		result = append(result, dto.NoticeResponse{
			ID:          n.ID,
			Title:       n.Title,
			Content:     n.Content,
			ReleaseTime: *n.ReleaseTime,
			// Status:      map[int]string{1: "有效", 0: "无效"}[n.Status],
		})
	}

	// 返回分页结果
	utils.SuccessPage(ctx, total, page, pageSize, result)
}

// GetNoticeContent 获取公告内容
func (ctr *NoticeController) GetNoticeContent(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.NoticeContentRequest
	if !utils.BindUrl(ctx, &req) {
		return
	}

	// 调用服务层
	notice, err := ctr.noticeService.GetNoticeContent(ctx, req.ID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	result := dto.NoticeContentResponse{
		ID:          notice.ID,
		Title:       notice.Title,
		Content:     notice.Content,
		ReleaseTime: *notice.ReleaseTime,
	}

	// 返回成功响应
	utils.Success(ctx, "success", result)
}
