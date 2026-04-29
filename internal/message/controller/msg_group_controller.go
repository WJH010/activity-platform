package controller

import (
	"activity-platform/internal/message/dto"
	"activity-platform/internal/message/model"
	"activity-platform/internal/message/service"
	"activity-platform/internal/utils"

	"github.com/gin-gonic/gin"
)

type MsgGroupController struct {
	msgGroupService service.MsgGroupService
}

func NewMsgGroupController(msgGroupService service.MsgGroupService) *MsgGroupController {
	return &MsgGroupController{msgGroupService: msgGroupService}
}

// GetMsgGroupByID 根据id获取消息群组信息
func (ctr *MsgGroupController) GetMsgGroupByID(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	// 调用服务层
	msgGroup, err := ctr.msgGroupService.GetMsgGroupByID(ctx, urlReq.MsgGroupID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回成功响应
	utils.Success(ctx, "success", msgGroup)
}

// AddUserToGroup 用户入群
func (ctr *MsgGroupController) AddUserToGroup(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定查询参数
	var req dto.UserListForGroupRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}
	// 获取当前登录userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 调用服务层
	err = ctr.msgGroupService.AddUserToGroup(ctx, urlReq.MsgGroupID, req.UserIDs, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回成功响应
	utils.Success(ctx, "success", nil)
}

// CreateMsgGroup 创建消息群组
func (ctr *MsgGroupController) CreateMsgGroup(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.CreateMsgGroupRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}
	// 获取当前登录userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 构建消息群组模型
	msgGroup := &model.UserMessageGroup{
		GroupName:      req.GroupName,
		Desc:           req.Desc,
		IncludeAllUser: req.IncludeAllUser,
		CreateUser:     userID,
		UpdateUser:     userID,
	}
	// 调用服务层
	err = ctr.msgGroupService.CreateMsgGroup(ctx, msgGroup, req.UserIDs)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回成功响应
	result := gin.H{
		"group_id": msgGroup.ID,
	}

	utils.Success(ctx, "success", result)
}

// DeleteUserFromGroup 用户退群
func (ctr *MsgGroupController) DeleteUserFromGroup(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定查询参数
	var req dto.UserListForGroupRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}
	// 获取当前登录userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 调用服务层
	err = ctr.msgGroupService.DeleteUserFromGroup(ctx, urlReq.MsgGroupID, req.UserIDs, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回成功响应
	utils.Success(ctx, "success", nil)
}

// UpdateMsgGroup 更新消息群组
func (ctr *MsgGroupController) UpdateMsgGroup(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定查询参数
	var req dto.UpdateMsgGroupRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}
	// 获取当前登录userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 调用服务层
	err = ctr.msgGroupService.UpdateMsgGroup(ctx, urlReq.MsgGroupID, req, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回成功响应
	utils.Success(ctx, "success", nil)
}

// DeleteMsgGroup 删除消息群组
func (ctr *MsgGroupController) DeleteMsgGroup(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 获取当前登录userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 调用服务层
	err = ctr.msgGroupService.DeleteMsgGroup(ctx, urlReq.MsgGroupID, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回成功响应
	utils.Success(ctx, "success", nil)
}

// ListMsgGroups 获取消息群组列表
func (ctr *MsgGroupController) ListMsgGroups(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.ListMsgGroupRequest
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
	result, total, err := ctr.msgGroupService.ListMsgGroups(ctx, page, pageSize, req)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回分页结果
	utils.SuccessPage(ctx, total, page, pageSize, result)
}

// ListGroupsUsers 获取消息群组用户列表
func (ctr *MsgGroupController) ListGroupsUsers(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定查询参数
	var req dto.ListPageRequest
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
	users, total, err := ctr.msgGroupService.ListGroupsUsers(ctx, page, pageSize, urlReq.MsgGroupID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回分页结果
	utils.SuccessPage(ctx, total, page, pageSize, users)
}

// ListNotInGroupUsers 获取不在指定组内的用户
func (ctr *MsgGroupController) ListNotInGroupUsers(ctx *gin.Context) {
	// 初始化参数结构体并绑定URL路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定查询参数
	var req dto.ListNotInGroupUsersRequest
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
	users, total, err := ctr.msgGroupService.ListNotInGroupUsers(ctx, page, pageSize, urlReq.MsgGroupID, req)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}
	// 返回分页结果
	utils.SuccessPage(ctx, total, page, pageSize, users)
}
