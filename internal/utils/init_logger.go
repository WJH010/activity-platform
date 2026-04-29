package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

// 定义ANSI颜色代码
const (
	// 重置所有样式
	colorReset = "\033[0m"

	// 文本颜色
	textBlack   = "\033[30m"
	textRed     = "\033[31;1m"
	textGreen   = "\033[32;1m"
	textYellow  = "\033[33;1m"
	textBlue    = "\033[34;1m"
	textMagenta = "\033[35m"
	textGray    = "\033[90m"
	textCyan    = "\033[36m"
	textWhite   = "\033[37m"
)

// 定义固定的字段顺序
var fieldOrder = []string{"method", "status", "latency", "path", "ip", "error"}

// 自定义格式化器
type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 时间戳染色
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	colorizedTime := fmt.Sprintf("%s%s%s", textWhite, timestamp, colorReset)

	// 日志级别染色
	var levelColor string
	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = textBlue
	case logrus.InfoLevel:
		levelColor = textGreen
	case logrus.WarnLevel:
		levelColor = textYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = textRed
	default:
		levelColor = entry.Level.String()
	}
	colorizedLevel := fmt.Sprintf("%s[%s]%s", levelColor, entry.Level.String(), colorReset)

	// 消息内容染色（这里使用默认颜色）
	colorizedMessage := entry.Message

	// 按固定顺序处理字段，对字段名，字段值自定义染色
	fields := ""
	for _, key := range fieldOrder {
		// 检查字段是否存在
		value, exists := entry.Data[key]
		if !exists {
			continue // 跳过不存在的字段
		}
		// 字段名染色
		keyStr := fmt.Sprintf("%s%s%s=", textGray, key, colorReset)

		// 根据字段名和值进行特定染色
		var valueStr string
		switch key {
		case "method":
			// 对HTTP方法染色
			method := fmt.Sprintf("%v", value)
			valueStr = fmt.Sprintf("%s%s%s", textBlue, method, colorReset)
		case "status":
			// 对状态码染色
			statusStr := fmt.Sprintf("%v", value)
			statusCode, _ := strconv.Atoi(statusStr)
			// 根据不同的值进行染色
			if statusCode >= 200 && statusCode < 300 {
				valueStr = fmt.Sprintf("%s%s%s", textGreen, statusStr, colorReset)
			} else if statusCode >= 400 && statusCode < 500 {
				valueStr = fmt.Sprintf("%s%s%s", textYellow, statusStr, colorReset)
			} else if statusCode >= 500 {
				valueStr = fmt.Sprintf("%s%s%s", textRed, statusStr, colorReset)
			} else {
				valueStr = statusStr
			}
		default:
			// 其他字段值用默认颜色
			valueStr = fmt.Sprintf("%v", value)
		}

		fields += fmt.Sprintf(" %s%s |", keyStr, valueStr)
	}

	// 组合所有部分
	logLine := fmt.Sprintf("%s %s %s%s\n",
		colorizedTime,
		colorizedLevel,
		colorizedMessage,
		fields)

	return []byte(logLine), nil
}

// InitLogger 初始化日志记录器,使用 logrus 作为日志记录器
func InitLogger() {
	// 使用自定义格式化器
	logrus.SetFormatter(&CustomFormatter{})

	// 创建日志目录
	os.MkdirAll("../logs/info", 0755)
	os.MkdirAll("../logs/error", 0755)

	infoLogPath := "../logs/info/info.log"
	infoWriter, err := rotatelogs.New(
		infoLogPath+".%Y%m%d",
		rotatelogs.WithLinkName(infoLogPath),
		rotatelogs.WithMaxAge(7*24*time.Hour),     // 保留7天
		rotatelogs.WithRotationTime(24*time.Hour), // 每天分割
	)
	if err != nil {
		logrus.Fatalf("配置 Info 日志分割器失败: %v", err)
	}

	// 创建 Error 级别日志的分割器（处理 Error、Fatal 和 Panic 级别）
	errLogPath := "../logs/error/err.log"
	errorWriter, err := rotatelogs.New(
		errLogPath+".%Y%m%d",
		rotatelogs.WithLinkName(errLogPath),
		rotatelogs.WithMaxAge(30*24*time.Hour),    // 保留30天
		rotatelogs.WithRotationTime(24*time.Hour), // 每天分割
	)
	if err != nil {
		logrus.Fatalf("配置 Error 日志分割器失败: %v", err)
	}

	// 添加日志钩子
	logrus.AddHook(lfshook.NewHook(
		// 定义不同日志级别对应的输出位置
		lfshook.WriterMap{
			logrus.InfoLevel:  infoWriter,
			logrus.WarnLevel:  infoWriter,
			logrus.ErrorLevel: errorWriter,
			logrus.FatalLevel: errorWriter,
			logrus.PanicLevel: errorWriter,
		},
		// 设置日志格式为 JSON 格式，并指定时间戳的格式为 RFC3339 标准
		// 此处也可使用自定义格式化器
		&logrus.JSONFormatter{TimestampFormat: time.RFC3339},
	))
	// 设置 logrus 为默认日志记录器
	log.SetOutput(logrus.StandardLogger().Writer())
	gin.DefaultWriter = logrus.StandardLogger().Writer()
}
