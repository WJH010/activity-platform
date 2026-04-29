package middleware

import (
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Recovery 恢复中间件（优化版：只记录业务代码关键错误位置）
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 捕获完整堆栈信息
				stack := make([]byte, 4096)
				length := runtime.Stack(stack, false) // 只获取当前协程堆栈（更轻量）
				stackStr := string(stack[:length])

				// 分割堆栈为行
				lines := strings.Split(stackStr, "\n")
				var filteredLines []string

				// 过滤规则：只保留包含项目业务代码路径的行（根据实际项目路径调整关键字）
				// 例如项目代码路径包含 "news-release-platform/internal/"，则保留相关行
				for i, line := range lines {
					// 保留业务代码调用栈（路径包含项目内部包路径）
					if strings.Contains(line, "news-release-platform/internal/") {
						// 同时保留函数名行和对应的文件行（通常是相邻两行）
						filteredLines = append(filteredLines, lines[i])
						if i+1 < len(lines) {
							filteredLines = append(filteredLines, lines[i+1])
						}
					}
				}

				// 拼接过滤后的关键堆栈信息
				cleanStack := strings.Join(filteredLines, "\n")

				// 记录关键错误信息
				logrus.WithFields(logrus.Fields{
					"panic": err,
					"stack": cleanStack, // 只包含业务代码的错误位置
				}).Error("发生panic")

				// 返回500错误
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "服务器内部错误",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
