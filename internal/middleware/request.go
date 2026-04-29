// internal/middleware/request.go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIdMiddleware 全局注入 RequestId
func RequestIdMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 生成/获取 requestId
		requestId := ctx.GetHeader("X-Request-Id")
		if requestId == "" {
			// 若请求头中没有传入requestId则生成一个
			requestId = uuid.NewString()
		}

		// 存入上下文，便于后续使用
		ctx.Set("requestId", requestId)

		// 响应头返回 requestId，便于前端获取
		ctx.Writer.Header().Set("X-Request-Id", requestId)

		// 继续执行后续中间件/接口
		ctx.Next()
	}
}
