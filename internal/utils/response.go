package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应结构体
type Response struct {
	Code      int    `json:"code"`      // 业务码（200=成功，其他=错误）
	Message   string `json:"message"`   // 提示信息
	Data      any    `json:"data"`      // 业务数据（成功时返回，错误时为null）
	RequestId string `json:"requestId"` // 请求ID（用于问题排查，可选）
}

// 分页响应结构体（扩展成功响应，用于列表接口）
type PageResponse struct {
	Total    int64 `json:"total"`    // 总条数
	Page     int   `json:"page"`     // 当前页
	PageSize int   `json:"pageSize"` // 每页条数
	List     any   `json:"list"`     // 当前页数据
}

// 获取 RequestId
func getRequestId(ctx *gin.Context) string {
	requestId, _ := ctx.Get("requestId")
	return requestId.(string)
}

// Success 通用成功响应
// 参数：ctx（Gin 上下文）、message（提示信息）、data（业务数据）
func Success(ctx *gin.Context, message string, data any) {
	// 若 message 为空，默认填充 "success"
	if message == "" {
		message = "success"
	}

	ctx.JSON(http.StatusOK, Response{
		Code:      200, // 成功业务码固定为 200
		Message:   message,
		Data:      data,
		RequestId: getRequestId(ctx),
	})
}

// SuccessPage 分页成功响应
// 参数：ctx、total（总条数）、page（当前页）、pageSize（每页条数）、list（当前页数据）
func SuccessPage(ctx *gin.Context, total int64, page, pageSize int, list any) {
	// 构造分页数据
	pageData := PageResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     list,
	}

	// 复用通用成功响应，将分页数据作为 Data 传入
	Success(ctx, "success", pageData)
}
