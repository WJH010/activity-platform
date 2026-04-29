package controller

import (
	"fmt"

	"activity-platform/internal/user/dto"
	"activity-platform/internal/user/service"
	"activity-platform/internal/utils"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
type UserController struct {
	userService service.UserService
}

// NewUserController 创建用户控制器实例
func NewUserController(userService service.UserService) *UserController {
	return &UserController{userService: userService}
}

// Login 微信登录接口
func (ctr *UserController) Login(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.WxLoginRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	accessToken, refreshToken, sessionSign, err := ctr.userService.Login(ctx, req.Code, req.EncryptedData, req.IV)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	if accessToken == "" || refreshToken == "" {
		err = utils.NewSystemError(fmt.Errorf("token生成异常"))
		utils.HandlerFunc(ctx, err)

		return
	}

	// ctx.JSON(http.StatusOK, gin.H{
	// 	"code":          200,
	// 	"message":       "登录成功",
	// 	"access_token":  accessToken,
	// 	"refresh_token": refreshToken,
	// 	"session_sign":  sessionSign,
	// })

	result := gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"session_sign":  sessionSign,
	}

	utils.Success(ctx, "success", result)
}

// RefreshToken 刷新token接口
func (ctr *UserController) RefreshToken(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.RefreshTokenRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	accessToken, refreshToken, err := ctr.userService.RefreshToken(ctx, req.RefreshToken, req.SessionSign)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	result := gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}

	utils.Success(ctx, "success", result)
}

// UpdateUserInfo 更新用户信息接口
func (ctr *UserController) UpdateUserInfo(ctx *gin.Context) {
	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 绑定并验证请求参数
	var req dto.UserUpdateRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 调用服务更新用户信息
	err = ctr.userService.UpdateUserInfo(ctx, userID, req)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// GetUserInfo 获取用户信息接口
func (ctr *UserController) GetUserInfo(ctx *gin.Context) {
	// 获取userID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务获取用户信息
	user, err := ctr.userService.GetUserByID(ctx, userID)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", user)
}

// ListAllUsers 列出所有用户接口（管理员权限）
func (ctr *UserController) ListAllUsers(ctx *gin.Context) {
	// 绑定并验证请求参数
	var req dto.ListUsersRequest
	if !utils.BindQuery(ctx, &req) {
		return
	}

	// 设置默认分页参数
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 调用服务获取用户列表
	users, total, err := ctr.userService.ListAllUsers(ctx, req.Page, req.PageSize, req)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.SuccessPage(ctx, total, req.Page, req.PageSize, users)
}

// BgLogin 后台登录
func (ctr *UserController) BgLogin(ctx *gin.Context) {
	// 绑定并验证请求参数
	var req dto.BgLoginRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	accessToken, refreshToken, err := ctr.userService.BgLogin(ctx, req)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	result := gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}

	utils.Success(ctx, "success", result)
}

// RefreshToken 刷新token接口
func (ctr *UserController) BgRefreshToken(ctx *gin.Context) {
	// 初始化参数结构体并绑定查询参数
	var req dto.RefreshTokenRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	accessToken, refreshToken, err := ctr.userService.BgRefreshToken(ctx, req.RefreshToken)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	result := gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}

	utils.Success(ctx, "success", result)
}

// CreateAdminUser 新增管理员
func (ctr *UserController) CreateAdminUser(ctx *gin.Context) {
	// 绑定并验证请求参数
	var req dto.CreateAdminRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取当前登录用户ID
	operator, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务新增管理员
	err = ctr.userService.CreateAdminUser(ctx, req, operator)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// UpdateAdminUser 更新管理员
func (ctr *UserController) UpdateAdminUser(ctx *gin.Context) {
	// 从路径参数获取userID
	var urlReq dto.UserIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	// 绑定并验证请求参数
	var req dto.UpdateAdminRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取当前登录用户ID
	operator, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务更新管理员
	err = ctr.userService.UpdateAdminUser(ctx, urlReq.UserID, req, operator)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

// UpdateAdminStatus 更新管理员状态
func (ctr *UserController) UpdateAdminStatus(ctx *gin.Context) {
	// 从路径参数获取userID
	var urlReq dto.UserIDRequest
	if !utils.BindUrl(ctx, &urlReq) {
		return
	}

	// 绑定并验证请求参数
	var req dto.UpdateAdminStatusRequest
	if !utils.BindJSON(ctx, &req) {
		return
	}

	// 获取当前登录用户ID
	operator, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务更新管理员状态
	err = ctr.userService.UpdateAdminStatus(ctx, urlReq.UserID, req.Operation, operator)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}

func (ctr *UserController) Logout(ctx *gin.Context) {
	// 获取当前登录用户ID
	userID, err := utils.GetUserID(ctx)
	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	// 调用服务处理退出登录
	err = ctr.userService.WxLogout(ctx, userID)

	// 处理异常
	if err != nil {
		utils.HandlerFunc(ctx, err)
		return
	}

	utils.Success(ctx, "success", nil)
}
