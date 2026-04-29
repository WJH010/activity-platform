package semantic_search

import (
	"encoding/json"
	"fmt"
	"strings"

	_ "embed"

	"activity-platform/internal/agent/rag/search"
	"activity-platform/internal/agent/skill"
)

//go:embed skill.md
var skillMD string

type semanticSearchSkill struct {
	meta      skill.SkillMeta
	searchSvc *search.SearchService
}

func New(searchSvc *search.SearchService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析semantic_search/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &semanticSearchSkill{
		meta:      meta,
		searchSvc: searchSvc,
	}
}

func (s *semanticSearchSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *semanticSearchSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	query, _ := skill.GetParamString(ctx.Params, "query")
	if query == "" {
		return skill.SkillResult{
			Success: false,
			Message: "请提供搜索关键词(query)",
		}, nil
	}

	searchTypeStr, _ := skill.GetParamString(ctx.Params, "search_type")
	topK := skill.GetParamIntWithDefault(ctx.Params, "top_k", 5)

	// 解析检索类型
	var searchType search.SearchType
	switch strings.ToLower(searchTypeStr) {
	case "article", "文章":
		searchType = search.SearchTypeArticle
	case "event", "活动":
		searchType = search.SearchTypeEvent
	default:
		searchType = search.SearchTypeAll
	}

	if topK < 1 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	// 执行混合检索
	resp, err := s.searchSvc.Search(ctx.Ctx, query, searchType, topK)
	if err != nil {
		return skill.SkillResult{}, fmt.Errorf("语义检索失败: %w", err)
	}

	if resp.Total == 0 {
		return skill.SkillResult{
			Success: true,
			Data:    map[string]interface{}{"results": []interface{}{}, "total": 0},
			Message: fmt.Sprintf("未找到与 \"%s\" 相关的内容", query),
		}, nil
	}

	// 聚合去重
	aggregated := search.DeduplicateAndAggregate(resp.Results)

	data, _ := json.Marshal(map[string]interface{}{
		"total":   len(aggregated),
		"query":   query,
		"results": aggregated,
	})

	sourceDesc := "文章和活动"
	if searchType == search.SearchTypeArticle {
		sourceDesc = "文章"
	} else if searchType == search.SearchTypeEvent {
		sourceDesc = "活动"
	}

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: fmt.Sprintf("在%s中语义检索到 %d 条与 \"%s\" 相关的结果", sourceDesc, len(aggregated), query),
	}, nil
}
