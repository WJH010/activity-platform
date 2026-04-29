package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求处理前：记录开始时间和基础信息
		start := time.Now()
		path := c.Request.URL.Path    // 请求路径
		raw := c.Request.URL.RawQuery // 查询参数

		// 执行后续中间件和业务逻辑
		c.Next()

		// 请求处理后：计算耗时并收集响应信息
		latency := time.Since(start)                                   // 计算请求处理耗时
		clientIP := c.ClientIP()                                       // 客户端IP
		method := c.Request.Method                                     // 请求方法
		statusCode := c.Writer.Status()                                // 响应状态码
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String() // 错误信息

		// 处理URL查询参数（非空时添加）
		if raw != "" {
			path = path + "?" + raw
		}

		// 构建日志字段
		entry := logrus.WithFields(logrus.Fields{
			"status":  statusCode,
			"method":  method,
			"path":    path,
			"ip":      clientIP,
			"latency": latency,
			"error":   errorMessage,
		})

		// 根据状态码分级输出日志
		if statusCode >= http.StatusInternalServerError {
			entry.Error("HTTP请求错误")
		} else if statusCode >= http.StatusBadRequest {
			entry.Warn("HTTP请求警告")
		} else {
			entry.Info("HTTP请求信息")
		}
	}
}
