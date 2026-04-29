package utils

import (
	"github.com/gin-gonic/gin"
)

func GetUserID(ctx *gin.Context) (int, error) {
	// 获取userID
	userID, exists := ctx.Get("userid")
	if !exists {
		return 0, NewBusinessError(ErrCodeGetUserIDFailed, "获取用户ID失败")
	}
	// 类型转换
	uid, ok := userID.(int)
	if !ok {
		return 0, NewBusinessError(ErrCodeGetUserIDFailed, "用户ID类型转换失败")
	}
	return uid, nil
}
