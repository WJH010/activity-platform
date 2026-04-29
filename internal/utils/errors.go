// Package utils 用于自定义错误类型
package utils

import "errors"

// BusinessError 业务异常（向用户展示,携带错误码和消息）
// 程序中的逻辑错误，如用户输入错误、资源未找到等。向用户展示具体的错误信息。
type BusinessError struct {
	Code    int    // 自定义错误码（对应utils中的错误码）
	Message string // 错误消息
}

func (e *BusinessError) Error() string {
	return e.Message
}

// NewBusinessError 创建业务异常
func NewBusinessError(code int, message string) error {
	return &BusinessError{
		Code:    code,
		Message: message,
	}
}

// GetBusinessError 提取BusinessError（用于获取错误码）
func GetBusinessError(err error) (*BusinessError, bool) {
	var e *BusinessError
	ok := errors.As(err, &e)
	return e, ok
}

// SystemError 系统异常（内部错误，不向用户展示详情）
// 程序中的系统错误，如数据库连接失败、服务器异常等。用于记录日志，向用户展示通用的错误信息。
// 这些错误通常是由于系统故障或配置问题引起的，需要管理员干预解决。
type SystemError struct {
	Err error // 原始错误（用于日志）
}

// 返回统一的错误消息
func (e *SystemError) Error() string {
	return "服务器内部错误"
}

// NewSystemError 创建系统异常
func NewSystemError(err error) error {
	return &SystemError{Err: err}
}
