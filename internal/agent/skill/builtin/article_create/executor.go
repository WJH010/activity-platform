package article_create

import (
	"fmt"
	"time"

	_ "embed"

	"activity-platform/internal/agent/skill"
	"activity-platform/internal/article/model"
	articleSvc "activity-platform/internal/article/service"
)

//go:embed skill.md
var skillMD string

type articleCreateSkill struct {
	meta       skill.SkillMeta
	articleSvc articleSvc.ArticleService
}

func New(articleSvc articleSvc.ArticleService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析article_create/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &articleCreateSkill{
		meta:       meta,
		articleSvc: articleSvc,
	}
}

func (s *articleCreateSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *articleCreateSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	articleTitle, ok := skill.GetParamString(ctx.Params, "article_title")
	if !ok || articleTitle == "" {
		return skill.SkillResult{
			Success: false,
			Message: "缺少必要参数: article_title",
		}, nil
	}

	articleType, ok := skill.GetParamString(ctx.Params, "article_type")
	if !ok || articleType == "" {
		return skill.SkillResult{
			Success: false,
			Message: "缺少必要参数: article_type",
		}, nil
	}

	articleContent, ok := skill.GetParamString(ctx.Params, "article_content")
	if !ok || articleContent == "" {
		return skill.SkillResult{
			Success: false,
			Message: "缺少必要参数: article_content",
		}, nil
	}

	briefContent, _ := skill.GetParamString(ctx.Params, "brief_content")
	// 如果未提供摘要，从正文截取前200字
	if briefContent == "" && len(articleContent) > 200 {
		briefContent = articleContent[:200] + "..."
	} else if briefContent == "" {
		briefContent = articleContent
	}

	fieldType, _ := skill.GetParamString(ctx.Params, "field_type")
	articleSource, _ := skill.GetParamString(ctx.Params, "article_source")
	coverImageURL, _ := skill.GetParamString(ctx.Params, "cover_image_url")
	isSelection := skill.GetParamIntWithDefault(ctx.Params, "is_selection", 0)

	if ctx.UserID == 0 {
		return skill.SkillResult{
			Success: false,
			Message: "请先登录后再创建文章",
		}, nil
	}

	article := &model.Article{
		ArticleTitle:   articleTitle,
		ArticleType:    articleType,
		ArticleContent: articleContent,
		ReleaseTime:    time.Now(),
		BriefContent:   briefContent,
		FieldType:      fieldType,
		ArticleSource:  articleSource,
		CoverImageURL:  coverImageURL,
		IsSelection:    isSelection,
		CreateUser:     ctx.UserID,
	}

	err := s.articleSvc.CreateArticle(ctx.Ctx, article, nil)
	if err != nil {
		return skill.SkillResult{
			Success: false,
			Message: fmt.Sprintf("创建文章失败: %s", err.Error()),
		}, nil
	}

	return skill.SkillResult{
		Success: true,
		Data: map[string]interface{}{
			"article_id": article.ArticleID,
			"title":      articleTitle,
		},
		Message: fmt.Sprintf("文章【%s】创建成功！", articleTitle),
	}, nil
}
