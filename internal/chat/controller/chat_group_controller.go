package controller

import (
	"github.com/gin-gonic/gin"

	"activity-platform/internal/chat/dto"
	"activity-platform/internal/chat/model"
	"activity-platform/internal/chat/service"
	"activity-platform/internal/utils"
)

// ChatGroupController 负责群组相关的 HTTP 请求
type ChatGroupController struct {
	groupService service.ChatGroupService
}

// NewChatGroupController 创建实例
func NewChatGroupController(groupService service.ChatGroupService) *ChatGroupController {
	return &ChatGroupController{groupService: groupService}
}

// CreateGroup 创建聊天群组
func (ctr *ChatGroupController) CreateGroup(c *gin.Context) {
	// 1. 绑定并验证请求参数
	var req dto.CreateGroupReq
	if !utils.BindJSON(c, &req) {
		// BindJSON 内部已经处理了错误响应
		return
	}

	// 2. 获取当前用户ID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	chatGroup := &model.ChatGroup{
		GroupName: req.GroupName,
		Desc:      req.Desc,
	}

	// 3. 调用 Service 创建群组
	group, err := ctr.groupService.CreateGroup(c, chatGroup, userID)
	if err != nil {
		// Service 层返回的错误，统一由 HandlerFunc 处理
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 构造并返回成功的响应
	resp := &dto.GroupInfoResp{
		ID:        group.ID,
		GroupName: group.GroupName,
		Desc:      group.Desc,
		OwnerID:   group.OwnerID,
	}
	utils.Success(c, "创建成功", resp)
}

// AddMembers godoc
// @Summary      添加群组成员
// @Description  向指定群组批量添加新成员
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        groupId   path      int  true  "群组ID"
// @Param        members_info   body      dto.AddMembersReq  true  "成员信息，可附带 with_history 参数 (Y/N) 控制历史消息"
// @Success      200  {object}  utils.Response "成功响应"
// @Failure      400  {object}  utils.Response "请求参数错误"
// @Failure      401  {object}  utils.Response "用户未登录"
// @Failure      403  {object}  utils.Response "无权限操作"
// @Failure      404  {object}  utils.Response "群组不存在"
// @Failure      500  {object}  utils.Response "服务器内部错误"
// @Router       /chat/groups/{groupId}/members [post]
func (ctr *ChatGroupController) AddMembers(c *gin.Context) {
	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}

	// 2. 绑定请求体中的 userIDs
	var addMembersReq dto.AddMembersReq
	if !utils.BindJSON(c, &addMembersReq) {
		return
	}

	// 3. 获取当前操作者ID
	operatorID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 调用 Service 添加成员
	err = ctr.groupService.AddMembers(c, groupIDReq.GroupID, &addMembersReq, operatorID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	utils.Success(c, "添加成功", nil)
}

// RemoveMembers godoc
// @Summary      移除群组成员
// @Description  从指定群组批量移除成员
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        groupId   path      int  true  "群组ID"
// @Param        members_info   body      dto.RemoveMembersReq  true  "要移除的成员信息"
// @Success      200  {object}  utils.Response "成功响应"
// @Failure      400  {object}  utils.Response "请求参数错误"
// @Failure      401  {object}  utils.Response "用户未登录"
// @Failure      403  {object}  utils.Response "无权限操作"
// @Failure      404  {object}  utils.Response "群组不存在"
// @Failure      500  {object}  utils.Response "服务器内部错误"
// @Router       /chat/groups/{groupId}/members [delete]
func (ctr *ChatGroupController) RemoveMembers(c *gin.Context) {
	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}

	// 2. 绑定请求体中的 userIDs
	var removeMembersReq dto.RemoveMembersReq
	if !utils.BindJSON(c, &removeMembersReq) {
		return
	}

	// 3. 获取当前操作者ID
	operatorID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 调用 Service 移除成员
	err = ctr.groupService.RemoveMembers(c, groupIDReq.GroupID, removeMembersReq.UserIDs, operatorID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	utils.Success(c, "移除成功", nil)
}

// ListGroupMembers godoc
// @Summary      查询群组成员列表
// @Description  分页查询指定群组的成员列表
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Param        groupId   path      int  true  "群组ID"
// @Param        page      query     int  false "页码"
// @Param        pageSize  query     int  false "每页数量"
// @Success      200  {object}  utils.PageResponse{data=[]dto.ListGroupMembersResponse} "成功响应"
// @Failure      400  {object}  utils.Response "请求参数错误"
// @Failure      401  {object}  utils.Response "用户未登录"
// @Failure      500  {object}  utils.Response "服务器内部错误"
// @Router       /chat/groups/{groupId}/members [get]
func (ctr *ChatGroupController) ListGroupMembers(c *gin.Context) {
	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}

	// 2. 绑定分页参数
	var req dto.ListGroupMembersReq
	if !utils.BindQuery(c, &req) {
		return
	}

	// 3. 设置默认分页
	page := req.Page
	if page == 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// 4. 调用 Service 查询成员列表
	users, total, err := ctr.groupService.ListGroupMembers(c, groupIDReq.GroupID, page, pageSize)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 5. 返回分页结果
	utils.SuccessPage(c, total, page, pageSize, users)
}

// HasUnreadMessages godoc
// @Summary      检查是否存在未读消息
// @Description  检查当前用户在所有群聊中是否有任何未读消息
// @Tags         Chat
// @Produce      json
// @Success      200  {object}  utils.Response{data=string} "成功响应，data为'Y'表示有未读，'N'表示没有"
// @Failure      401  {object}  utils.Response "用户未登录"
// @Failure      500  {object}  utils.Response "服务器内部错误"
// @Router       /chat/unread [get]
func (ctr *ChatGroupController) HasUnreadMessages(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 2. 调用 Service 检查是否存在未读消息
	hasUnread, err := ctr.groupService.HasUnreadMessages(c, userID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 3. 根据结果返回 'Y' 或 'N'
	response := "N"
	if hasUnread {
		response = "Y"
	}

	utils.Success(c, "查询成功", response)
}

// ListUserGroups godoc
// @Summary      查询当前用户的群聊列表
// @Description  获取当前用户加入的所有群聊列表，包含每个群聊的最后一条消息和未读状态
// @Tags         Chat
// @Produce      json
// @Success      200  {object}  utils.Response{data=[]dto.ListUserGroupsResponse} "成功响应"
// @Failure      401  {object}  utils.Response "用户未登录"
// @Failure      500  {object}  utils.Response "服务器内部错误"
// @Router       /chat/groups [get]
func (ctr *ChatGroupController) ListUserGroups(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 2. 调用 Service 查询群组列表
	groups, err := ctr.groupService.ListUserGroups(c, userID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 3. 返回查询结果
	utils.Success(c, "查询成功", groups)
}

// ListGroupMessages godoc
// @Summary      查询指定群组的消息列表
// @Description  分页查询指定群组的消息，支持按内容和日期范围筛选，仅返回用户入群后的消息
// @Tags         Chat
// @Produce      json
// @Param        groupId   path      int  true  "群组ID"
// @Param        page      query     int  false "页码" default(1)
// @Param        pageSize  query     int  false "每页数量" default(10)
// @Param        content   query     string false "消息内容，模糊查询"
// @Param        startDate query     string false "开始日期 (格式: YYYY-MM-DD)"
// @Param        endDate   query     string false "结束日期 (格式: YYYY-MM-DD)"
// @Success      200  {object}  utils.PageResponse{data=[]dto.ListGroupMessagesResponse} "成功响应"
// @Failure      400  {object}  utils.Response "请求参数错误"
// @Failure      401  {object}  utils.Response "用户未登录"
// @Failure      403  {object}  utils.Response "无权限查看"
// @Failure      500  {object}  utils.Response "服务器内部错误"
// @Router       /chat/groups/{groupId}/messages [get]
func (ctr *ChatGroupController) ListGroupMessages(c *gin.Context) {
	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}

	// 2. 绑定分页和筛选参数
	var req dto.ListGroupMessagesReq
	if !utils.BindQuery(c, &req) {
		return
	}

	// 3. 获取当前用户ID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 设置默认分页
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 5. 调用 Service 查询消息列表
	messages, total, err := ctr.groupService.ListGroupMessages(c, groupIDReq.GroupID, userID, req)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 6. 返回分页结果
	utils.SuccessPage(c, total, req.Page, req.PageSize, messages)
}

// DeleteGroup 删除群聊
func (ctr *ChatGroupController) DeleteGroup(c *gin.Context) {
	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}

	// 2. 获取当前用户ID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 调用 Service 删除群聊
	err = ctr.groupService.DeleteGroup(c, groupIDReq.GroupID, userID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 5. 返回成功响应
	utils.Success(c, "删除群聊成功", nil)
}

func (ctr *ChatGroupController) ListAllGroups(c *gin.Context) {
	// 1. 绑定分页参数
	var req dto.ListGroupMembersReq
	if !utils.BindQuery(c, &req) {
		return
	}

	// 2. 设置默认分页
	page := req.Page
	if page == 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// 3. 调用 Service 查询群组列表
	groups, total, err := ctr.groupService.ListAllGroups(c, page, pageSize)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 返回分页结果
	utils.SuccessPage(c, total, page, pageSize, groups)
}

// GetUsersNotInGroup 获取不在指定群组中的用户列表
func (ctr *ChatGroupController) GetUsersNotInGroup(c *gin.Context) {
	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}

	var req dto.GetUsersNotInGroupRequest
	if !utils.BindQuery(c, &req) {
		return
	}

	// 3. 设置默认分页
	page := req.Page
	if page == 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 10
	}

	// 4. 调用 Service 查询
	users, total, err := ctr.groupService.FindUsersNotInGroup(c, groupIDReq.GroupID, page, pageSize, req.Name)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 5. 返回分页结果
	utils.SuccessPage(c, total, page, pageSize, users)
}
