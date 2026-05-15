package controller

import (
	"event-platform/internal/message/dto"
	"event-platform/internal/message/model"
	"event-platform/internal/message/service"
	"event-platform/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// MessageController 控制器
type MessageController struct {
	messageService service.MessageService
}

// NewMessageController 创建控制器实例
func NewMessageController(messageService service.MessageService) *MessageController {
	return &MessageController{messageService: messageService}
}

// GetMessageContent 获取消息内容
func (ctr *MessageController) GetMessageContent(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.MessageIDRequest
	if !utils.BindUrl(ctx, &req) {
		return
	}

	// 调用服务层
	message, err := ctr.messageService.GetMessageContent(ctx, req.MessageID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	result := dto.MessageContentResponse{
		ID:       message.ID,
		Title:    message.Title,
		Content:  message.Content,
		SendTime: message.SendTime,
	}

	// 返回成功响应
	utils.Success(ctx, "success", result)
}

// HasUnreadMessages 检查用户是否有未读消息
func (ctr *MessageController) HasUnreadMessages(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.HasUnreadMessagesRequest
	if !utils.BindQuery(ctx, &req) {
		return
	}

	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层
	hasUnread, err := ctr.messageService.HasUnreadMessages(ctx, userID, req.TypeCode)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回成功响应
	result := gin.H{
		"hasUnread": hasUnread,
	}

	utils.Success(ctx, "success", result)
}

// MarkAllMessagesAsRead 一键已读
func (ctr *MessageController) MarkAllMessagesAsRead(ctx *gin.Context) {
	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层
	err = ctr.messageService.MarkAllMessagesAsRead(ctx, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回成功响应
	utils.Success(ctx, "success", nil)
}

// ListUserMessageGroups 分页查询用户消息群组列表
func (ctr *MessageController) ListUserMessageGroups(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.ListUserGroupMessageRequest
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

	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层
	list, total, err := ctr.messageService.ListMessageGroupsByUserID(ctx, page, pageSize, userID, req.TypeCode)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.SuccessPage(ctx, total, page, pageSize, list)
}

// ListMsgByGroups 分页查询分组内消息列表
func (ctr *MessageController) ListMsgByGroups(ctx *gin.Context) {
	// 获取并绑定路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定查询参数
	var req dto.ListMessageByGroupRequest
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

	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层
	list, total, err := ctr.messageService.ListMsgByGroups(ctx, page, pageSize, urlReq.MsgGroupID, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.SuccessPage(ctx, total, page, pageSize, list)
}

// SendMessage 发送消息
func (ctr *MessageController) SendMessage(ctx *gin.Context) {
	// 获取并绑定路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}
	// 初始化参数结构体并绑定请求体参数
	var req dto.SendMessageRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 构建消息对象
	message := &model.Message{
		Title:      req.Title,
		Content:    req.Content,
		SendTime:   time.Now(),
		CreateUser: userID,
		UpdateUser: userID,
	}

	// 调用服务层
	err = ctr.messageService.SendMessage(ctx, urlReq.MsgGroupID, message)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回成功响应
	utils.Success(ctx, "success", nil)
}

// ListMessagesByGroupID 根据消息群组ID查询消息列表
func (ctr *MessageController) ListMessagesByGroupID(ctx *gin.Context) {
	// 获取并绑定路径参数
	var urlReq dto.MsgGroupIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	// 初始化参数结构体并绑定查询参数
	var req dto.ListMessageByGroupIDRequest
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
	list, total, err := ctr.messageService.ListMessagesByGroupID(ctx, page, pageSize, urlReq.MsgGroupID, req.Title, req.QueryScope)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.SuccessPage(ctx, total, page, pageSize, list)
}

// RevokeGroupMessage 撤回群组消息
func (ctr *MessageController) RevokeGroupMessage(ctx *gin.Context) {
	// 获取并绑定路径参数
	var urlReq dto.DeleteMsgGroupMapRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务层
	err = ctr.messageService.RevokeGroupMessage(ctx, urlReq.MapID, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 返回成功响应
	utils.Success(ctx, "success", nil)
}
