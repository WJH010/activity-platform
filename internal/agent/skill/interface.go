package skill

import (
	"context"
	"event-platform/internal/agent/llm"
	"fmt"
)

// SkillMeta 从skill.md解析出的Skill元数据
type SkillMeta struct {
	Name         string      `yaml:"name"`          // Skill唯一标识名，如 event_query
	DisplayName  string      `yaml:"display_name"`  // 前端展示名，如 活动查询
	Category     string      `yaml:"category"`      // 分类：event/article/user
	AuthRequired bool        `yaml:"auth_required"` // 是否需要登录
	Parameters   interface{} `yaml:"parameters"`    // JSON Schema 参数定义
	Description  string      `yaml:"-"`             // Markdown正文（LLM可读的详细描述）
	IsBuiltin    bool        `yaml:"-"`             // 是否为内置Skill
}

// SkillContext Skill执行上下文
type SkillContext struct {
	Ctx       context.Context        // 请求上下文
	Params    map[string]interface{} // LLM函数调用传入的参数
	UserID    int                    // 当前用户ID（需登录时有效）
	AuthToken string                 // JWT Token（用于内部API调用）
}

// SkillResult Skill执行结果
type SkillResult struct {
	Success bool        `json:"success"` // 是否成功
	Data    interface{} `json:"data"`    // 结果数据
	Message string      `json:"message"` // 给LLM的可读消息
}

// Skill Skill统一接口
type Skill interface {
	// Meta 返回Skill元数据
	Meta() SkillMeta
	// Execute 执行Skill
	Execute(ctx SkillContext) (SkillResult, error)
}

// ToToolDefinition 将SkillMeta转换为LLM的ToolDefinition
func (m SkillMeta) ToToolDefinition() llm.ToolDefinition {
	// 构建参数Schema，若为空则设置默认值
	params := m.Parameters
	if params == nil {
		params = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	return llm.ToolDefinition{
		Type: "function",
		Function: llm.FunctionDef{
			Name:        m.Name,
			Description: m.Description,
			Parameters:  params,
		},
	}
}

// --- 参数提取辅助函数 ---

// GetParamString 从参数map中获取字符串值
func GetParamString(params map[string]interface{}, key string) (string, bool) {
	val, ok := params[key]
	if !ok {
		return "", false
	}
	switch v := val.(type) {
	case string:
		return v, true
	case float64:
		return fmt.Sprintf("%.0f", v), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

// GetParamInt 从参数map中获取整数值
// JSON反序列化后数字类型为float64，需要转换
func GetParamInt(params map[string]interface{}, key string) (int, bool) {
	val, ok := params[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case string:
		// 尝试从字符串解析（某些LLM可能返回字符串形式的数字）
		var i int
		_, err := fmt.Sscanf(v, "%d", &i)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

// GetParamIntWithDefault 从参数map中获取整数值，带默认值
func GetParamIntWithDefault(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := GetParamInt(params, key); ok {
		return v
	}
	return defaultVal
}

// GetParamStringWithDefault 从参数map中获取字符串值，带默认值
func GetParamStringWithDefault(params map[string]interface{}, key string, defaultVal string) string {
	if v, ok := GetParamString(params, key); ok {
		return v
	}
	return defaultVal
}
