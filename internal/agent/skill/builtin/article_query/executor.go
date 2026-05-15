package article_query

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"event-platform/internal/agent/skill"
	articleSvc "event-platform/internal/article/service"
)

//go:embed skill.md
var skillMD string

type articleQuerySkill struct {
	meta       skill.SkillMeta
	articleSvc articleSvc.ArticleService
}

func New(articleSvc articleSvc.ArticleService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析article_query/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &articleQuerySkill{
		meta:       meta,
		articleSvc: articleSvc,
	}
}

func (s *articleQuerySkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *articleQuerySkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	page := skill.GetParamIntWithDefault(ctx.Params, "page", 1)
	pageSize := skill.GetParamIntWithDefault(ctx.Params, "page_size", 10)
	articleTitle, _ := skill.GetParamString(ctx.Params, "article_title")
	articleType, _ := skill.GetParamString(ctx.Params, "article_type")
	fieldType, _ := skill.GetParamString(ctx.Params, "field_type")

	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}

	articles, total, err := s.articleSvc.ListArticle(ctx.Ctx, page, pageSize, articleTitle, articleType, "", fieldType, 0, "")
	if err != nil {
		return skill.SkillResult{}, fmt.Errorf("查询文章列表失败: %w", err)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"articles":  articles,
	})

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: fmt.Sprintf("查询到 %d 篇文章，共 %d 篇", len(articles), total),
	}, nil
}
