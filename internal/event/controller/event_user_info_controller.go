package controller

import (
	"event-platform/internal/event/dto"
	"event-platform/internal/event/model"
	"event-platform/internal/event/service"
	"event-platform/internal/utils"

	"github.com/gin-gonic/gin"
)

// EventUserInfoController 控制器
type EventUserInfoController struct {
	eventUserInfoService service.EventUserInfoService
}

// NewEventUserInfoController 创建 EventUserInfoController 实例
func NewEventUserInfoController(eventUserInfoService service.EventUserInfoService) *EventUserInfoController {
	return &EventUserInfoController{eventUserInfoService: eventUserInfoService}
}

// Create 创建用户信息字段
func (ctr *EventUserInfoController) Create(ctx *gin.Context) {
	var req dto.CreateEventUserInfoRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	userField := &model.EventUserInfo{
		Code: req.Code,
		Name: req.Name,
	}

	if err := ctr.eventUserInfoService.Create(ctx, userField); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// Update 更新用户信息字段
func (ctr *EventUserInfoController) Update(ctx *gin.Context) {
	var urlReq dto.EventUserInfoIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	var req dto.UpdateEventUserInfoRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if err := ctr.eventUserInfoService.Update(ctx, urlReq.ID, req); err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// List 查询用户信息字段列表
func (ctrl *EventUserInfoController) List(ctx *gin.Context) {
	var req dto.ListEventUserInfoRequest
	if !utils.BindQuery(ctx, &req) {
		return
	}

	userFields, err := ctrl.eventUserInfoService.List(ctx, req)
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", userFields)
}

// UpdateStatus 更新用户信息字段状态
func (ctrl *EventUserInfoController) UpdateStatus(ctx *gin.Context) {
	var urlReq dto.EventUserInfoIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定请求体
	var req dto.UpdateEventUserInfoStatusRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	if req.IsDeleted == utils.DeletedFlagYes {
		if err := ctrl.eventUserInfoService.Delete(ctx, urlReq.ID); err != nil {
			utils.HandlerFunc(ctx, err)
			return
		}
	} else if req.IsDeleted == utils.DeletedFlagNo {
		if err := ctrl.eventUserInfoService.Restore(ctx, urlReq.ID); err != nil {
			utils.HandlerFunc(ctx, err)
			return
		}
	}

	utils.Success(ctx, "success", nil)
}
