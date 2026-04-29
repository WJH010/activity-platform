package skill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"activity-platform/internal/agent/model"

	"github.com/sirupsen/logrus"
)

// DynamicSkill 动态Skill（从数据库加载，通过HTTP模板执行）
type DynamicSkill struct {
	meta       SkillMeta
	definition model.SkillDefinition
	httpClient *http.Client
}

// NewDynamicSkill 从数据库SkillDefinition创建动态Skill
func NewDynamicSkill(def model.SkillDefinition) Skill {
	meta := SkillMeta{
		Name:         def.Name,
		DisplayName:  def.DisplayName,
		Category:     def.Category,
		AuthRequired: def.AuthRequired == 1,
		IsBuiltin:    false,
	}

	// 解析参数Schema
	if def.ParamSchema != "" {
		var params interface{}
		if err := json.Unmarshal([]byte(def.ParamSchema), &params); err != nil {
			logrus.Warnf("动态Skill [%s] 参数Schema解析失败: %v", def.Name, err)
		} else {
			meta.Parameters = params
		}
	}
	if meta.Parameters == nil {
		meta.Parameters = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// 使用Description字段作为LLM描述
	meta.Description = def.Description

	return &DynamicSkill{
		meta:       meta,
		definition: def,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (ds *DynamicSkill) Meta() SkillMeta {
	return ds.meta
}

func (ds *DynamicSkill) Execute(ctx SkillContext) (SkillResult, error) {
	// 渲染URL
	url, err := ds.renderTemplate("url", ds.definition.UrlTemplate, ctx.Params)
	if err != nil {
		return SkillResult{}, fmt.Errorf("渲染URL模板失败: %w", err)
	}

	// 渲染Headers
	var headers map[string]string
	if ds.definition.HeadersTemplate != "" {
		headers, err = ds.renderHeaders(ds.definition.HeadersTemplate, ctx.Params)
		if err != nil {
			return SkillResult{}, fmt.Errorf("渲染Headers模板失败: %w", err)
		}
	}

	// 渲染Body
	var body string
	if ds.definition.BodyTemplate != "" {
		body, err = ds.renderTemplate("body", ds.definition.BodyTemplate, ctx.Params)
		if err != nil {
			return SkillResult{}, fmt.Errorf("渲染Body模板失败: %w", err)
		}
	}

	// 创建HTTP请求
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx.Ctx, ds.definition.HttpMethod, url, reqBody)
	if err != nil {
		return SkillResult{}, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置Headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 注入Auth Token
	if ds.meta.AuthRequired && ctx.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+ctx.AuthToken)
	}

	// 发送请求
	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return SkillResult{}, fmt.Errorf("执行HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return SkillResult{}, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		logrus.Errorf("动态Skill [%s] HTTP请求失败, status=%d, body=%s",
			ds.meta.Name, resp.StatusCode, string(respBody))
		return SkillResult{
			Success: false,
			Data:    nil,
			Message: fmt.Sprintf("接口调用失败(HTTP %d): %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// 解析响应JSON
	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// 非JSON响应，直接返回文本
		return SkillResult{
			Success: true,
			Data:    string(respBody),
			Message: "接口调用成功",
		}, nil
	}

	return SkillResult{
		Success: true,
		Data:    result,
		Message: "接口调用成功",
	}, nil
}

// renderTemplate 使用Go text/template渲染模板
func (ds *DynamicSkill) renderTemplate(name, tplStr string, params map[string]interface{}) (string, error) {
	if tplStr == "" {
		return "", nil
	}

	tpl, err := template.New(name).Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return buf.String(), nil
}

// renderHeaders 渲染Headers模板
// Headers模板应为JSON格式: {"X-Custom-Header": "{{.header_value}}"}
func (ds *DynamicSkill) renderHeaders(tplStr string, params map[string]interface{}) (map[string]string, error) {
	rendered, err := ds.renderTemplate("headers", tplStr, params)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	if err := json.Unmarshal([]byte(rendered), &headers); err != nil {
		return nil, fmt.Errorf("Headers模板渲染结果不是有效JSON: %w", err)
	}

	return headers, nil
}
