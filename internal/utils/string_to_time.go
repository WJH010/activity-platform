package utils

import "time"

func StringToTime(timeStr string) (time.Time, error) {
	// 定义允许的时间格式
	allowedFormats := []string{
		"2006-01-02",          // 日期格式
		"2006-01-02 15:04:05", // 日期时间格式
	}

	var parsedTime time.Time
	var err error

	// 尝试用每种格式解析
	for _, format := range allowedFormats {
		parsedTime, err = time.ParseInLocation(format, timeStr, time.Local)
		if err == nil {
			return parsedTime, nil // 成功解析，返回结果
		}
	}

	return time.Time{}, NewBusinessError(ErrCodeParamTypeError, "日期格式错误，请使用 YYYY-MM-DD 或 YYYY-MM-DD HH:MM:SS 格式")
}
