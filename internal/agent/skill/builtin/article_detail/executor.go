package article_detail

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	_ "embed"

	"event-platform/internal/agent/skill"
	articleSvc "event-platform/internal/article/service"
)

//go:embed skill.md
var skillMD string

// maxArticleContentChars ArticleContent字段返回给LLM的最大字符数
// 约6K token安全线内，避免撑爆上下文
const maxArticleContentChars = 2000

type articleDetailSkill struct {
	meta       skill.SkillMeta
	articleSvc articleSvc.ArticleService
}

func New(articleSvc articleSvc.ArticleService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析article_detail/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &articleDetailSkill{
		meta:       meta,
		articleSvc: articleSvc,
	}
}

func (s *articleDetailSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *articleDetailSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	articleID, ok := skill.GetParamInt(ctx.Params, "article_id")
	if !ok {
		return skill.SkillResult{
			Success: false,
			Message: "缺少必要参数: article_id",
		}, nil
	}

	article, err := s.articleSvc.GetArticleContent(ctx.Ctx, articleID)
	if err != nil {
		return skill.SkillResult{
			Success: false,
			Message: fmt.Sprintf("查询文章详情失败: %s", err.Error()),
		}, nil
	}

	// 截断 ArticleContent 字段，保留完整的 BriefContent
	// 原始HTML内容可能非常长，直接传给LLM有上下文溢出风险
	truncatedContent := truncateRunes(article.ArticleContent, maxArticleContentChars)
	if utf8.RuneCountInString(article.ArticleContent) > maxArticleContentChars {
		truncatedContent += "\n...[正文过长已截断，如需完整内容请告知用户前往文章详情页查看]"
	}
	article.ArticleContent = truncatedContent

	data, _ := json.Marshal(article)

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: fmt.Sprintf("文章【%s】详情获取成功", article.ArticleTitle),
	}, nil
}

// truncateRunes 按字符数截断字符串
func truncateRunes(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}
