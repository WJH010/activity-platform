package skill

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseSkillMD 解析skill.md文件内容，返回SkillMeta
//
// skill.md格式：
//
//	---
//	name: event_query
//	display_name: 活动查询
//	category: event
//	auth_required: true
//	parameters:
//	  type: object
//	  properties:
//	    page:
//	      type: integer
//	      description: 页码
//	  required: []
//	---
//
//	Markdown正文，作为LLM可读的详细描述...
func ParseSkillMD(content string) (SkillMeta, error) {
	var meta SkillMeta

	// 检查是否以Frontmatter标记开头
	if !strings.HasPrefix(content, "---") {
		return meta, fmt.Errorf("skill.md必须以YAML Frontmatter (---) 开头")
	}

	// 找到Frontmatter结束标记
	// 跳过第一个 ---
	rest := content[3:]
	// 跳过换行
	rest = strings.TrimLeft(rest, "\r\n")

	// 找到第二个 ---
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return meta, fmt.Errorf("skill.md Frontmatter缺少结束标记 (---)")
	}

	frontmatter := rest[:endIdx]

	// 提取Markdown正文（结束标记之后的内容）
	bodyStart := endIdx + len("\n---")
	body := strings.TrimLeft(rest[bodyStart:], "\r\n ")
	meta.Description = strings.TrimSpace(body)

	// 解析YAML Frontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return meta, fmt.Errorf("解析skill.md Frontmatter失败: %w", err)
	}

	// 校验必要字段
	if meta.Name == "" {
		return meta, fmt.Errorf("skill.md缺少name字段")
	}
	if meta.DisplayName == "" {
		return meta, fmt.Errorf("skill.md缺少display_name字段")
	}
	if meta.Category == "" {
		return meta, fmt.Errorf("skill.md缺少category字段")
	}

	return meta, nil
}
