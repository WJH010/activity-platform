package utils

import (
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators 向现有验证器注册自定义规则
func RegisterCustomValidators(v *validator.Validate) {
	// 时间格式验证
	err := v.RegisterValidation("time_format", validateTimeFormat)
	if err != nil {
		panic("注册时间格式验证失败: " + err.Error())
	}
	// 用户昵称验证
	if err := v.RegisterValidation("nickname", validateNickname); err != nil {
		panic("注册昵称验证失败: " + err.Error())
	}

	// 真实姓名验证（支持中英文）
	if err := v.RegisterValidation("real_name", validateRealName); err != nil {
		panic("注册真实姓名验证失败: " + err.Error())
	}

	// 手机号验证
	if err := v.RegisterValidation("phone", validatePhoneNumber); err != nil {
		panic("注册手机号验证失败: " + err.Error())
	}

	// 非空字符串验证
	if err := v.RegisterValidation("non_empty_string", validateNonEmptyString); err != nil {
		panic("注册非空字符串验证失败: " + err.Error())
	}

	// 查询范围验证
	if err := v.RegisterValidation("query_scope", validateQueryScope); err != nil {
		panic("注册查询范围验证失败: " + err.Error())
	}
	// 用户群组消息类型验证
	if err := v.RegisterValidation("user_group_message_type", validateUserGroupMessageType); err != nil {
		panic("注册用户群组消息类型验证失败: " + err.Error())
	}

	// 其他自定义规则...
}

// 时间格式验证实现
func validateTimeFormat(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	if fieldValue == "" {
		return true // 空值通过（配合omitempty使用）
	}
	// 定义允许的时间格式
	allowedFormats := []string{
		"2006-01-02",          // 日期格式
		"2006-01-02 15:04:05", // 日期时间格式
	}

	// 尝试用每种格式解析
	for _, format := range allowedFormats {
		if _, err := time.Parse(format, fieldValue); err == nil {
			return true
		}
	}

	return false
}

// 用户昵称验证实现
func validateNickname(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	if fieldValue == "" {
		return true // 空值通过（配合omitempty使用）
	}
	// 检查长度
	if len(fieldValue) < 2 || len(fieldValue) > 20 {
		return false
	}
	// 检查是否包含非法字符（仅允许汉字、数字、大小写字母和下划线）
	match, _ := regexp.MatchString(`^[\p{Han}0-9a-zA-Z_]+$`, fieldValue)

	return match
}

// 姓名验证实现
func validateRealName(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	if fieldValue == "" {
		return true // 空值通过（配合omitempty使用）
	}
	// 检查长度 (1-50个字符)
	if len(fieldValue) < 1 || len(fieldValue) > 50 {
		return false
	}
	// 允许：汉字、英文字母、空格、点、连字符和单引号
	match, _ := regexp.MatchString(`^[\p{Han}a-zA-Z .'-]+$`, fieldValue)
	return match
}

// 手机号验证实现（中国大陆手机号）
func validatePhoneNumber(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	if fieldValue == "" {
		return true // 空值通过（配合omitempty使用）
	}
	// 中国大陆手机号规则：1开头，第二位3-9，后面9位数字
	match, _ := regexp.MatchString(`^1[3-9]\d{9}$`, fieldValue)
	return match
}

// 非空字符串验证实现
func validateNonEmptyString(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	if strings.TrimSpace(fieldValue) == "" {
		return false // 非空字符串验证失败
	}
	return true // 非空字符串验证成功
}

// 查询范围验证实现
func validateQueryScope(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()
	if fieldValue == "" {
		return true // 空值通过（配合omitempty使用）
	}
	// 允许的查询范围
	allowedScopes := QueryScopeList
	for _, scope := range allowedScopes {
		if strings.EqualFold(fieldValue, scope) {
			return true
		}
	}
	return false
}

// 用户群组消息类型验证实现
func validateUserGroupMessageType(fl validator.FieldLevel) bool {
	fieldValue := fl.Field().String()

	// 允许的消息类型
	allowedTypes := UserGroupMessageTypeList
	for _, t := range allowedTypes {
		if strings.EqualFold(fieldValue, t) {
			return true
		}
	}
	return false
}
