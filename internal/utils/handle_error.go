package utils

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RespondWithError 返回统一格式的JSON异常响应
//- ctx: Gin上下文
//- err: 错误对象
//- statusCode: HTTP状态码
//- code: 业务错误码
//- message: 用户友好的错误消息

func RespondWithError(ctx *gin.Context, err error, statusCode int, code int, message string) {
	// 记录错误日志，包含原始错误、HTTP状态码、业务错误码和提示信息
	logrus.WithError(err).
		WithField("status", statusCode).
		WithField("errorCode", code).
		Error(message)

	if ctx != nil {
		// 向客户端返回JSON格式的错误响应
		ctx.JSON(statusCode, Response{
			Code:      code,
			Message:   message,
			Data:      nil,
			RequestId: getRequestId(ctx),
		})
		ctx.Abort() // 终止后续处理
	}
}

// HandlerFunc 封装错误处理逻辑
// 注意：调用后需手动添加 return 终止当前函数，避免后续代码执行
func HandlerFunc(ctx *gin.Context, err error) {
	// 处理业务错误
	if bizErr, ok := GetBusinessError(err); ok {
		RespondWithError(ctx, err, http.StatusBadRequest, bizErr.Code, bizErr.Message)
		return
	}

	// 处理系统错误
	var sysErr *SystemError
	if errors.As(err, &sysErr) {
		RespondWithError(ctx, sysErr.Err, http.StatusInternalServerError, ErrCodeServerInternalError, err.Error())
		return
	}

	// 处理未知错误
	RespondWithError(ctx, err, http.StatusInternalServerError, ErrCodeServerInternalError, "未知服务器错误")
}

// IsUniqueConstraintError 判断是否为唯一索引冲突错误
// 返回值：(是否为唯一冲突, 冲突字段名或索引名)
func IsUniqueConstraintError(err error) (bool, string, string) {
	// 先判断是否是GORM的错误（如记录未找到等，但唯一冲突通常是底层错误）
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "", ""
	}

	// 适配MySQL
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1062 { // MySQL唯一索引冲突错误码
			name, value := parseMySQLUniqueField(mysqlErr.Message)
			return true, name, value
		}
	}

	// 可以在这里添加其他数据库类型的判断，如PostgreSQL、SQLite等

	return false, "", ""
}

// 解析MySQL错误信息中的冲突字段
func parseMySQLUniqueField(msg string) (string, string) {
	// 错误信息格式示例："Duplicate entry 'test' for key 'users.username'"

	// 提取 "users.username" 中的 "username" 部分
	start := strings.LastIndex(msg, ".")
	if start == -1 {
		return "unknown", ""
	}
	end := strings.LastIndex(msg, "'")
	if end == -1 || end <= start {
		return "unknown", ""
	}

	// 提取冲突值（entry后的内容）
	entryStart := strings.Index(msg, "Duplicate entry '")
	if entryStart == -1 {
		return "unknown", ""
	}
	entryStart += len("Duplicate entry '")

	entryEnd := strings.Index(msg[entryStart:], "'")
	if entryEnd == -1 {
		return "unknown", ""
	}
	value := msg[entryStart : entryStart+entryEnd]
	return msg[start+1 : end], value
}
