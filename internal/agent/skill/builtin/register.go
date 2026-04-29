package builtin

import (
	"activity-platform/internal/agent/rag/search"
	"activity-platform/internal/agent/skill"
	"activity-platform/internal/agent/skill/builtin/article_create"
	"activity-platform/internal/agent/skill/builtin/article_detail"
	"activity-platform/internal/agent/skill/builtin/article_query"
	"activity-platform/internal/agent/skill/builtin/event_cancel"
	"activity-platform/internal/agent/skill/builtin/event_detail"
	"activity-platform/internal/agent/skill/builtin/event_query"
	"activity-platform/internal/agent/skill/builtin/event_register"
	"activity-platform/internal/agent/skill/builtin/semantic_search"
	"activity-platform/internal/agent/skill/builtin/user_info"
	"activity-platform/internal/agent/skill/builtin/user_registered_events"

	articleSvc "activity-platform/internal/article/service"
	eventSvc "activity-platform/internal/event/service"
	userSvc "activity-platform/internal/user/service"
)

// Dependencies 内置Skill所需的依赖
type Dependencies struct {
	EventSvc   eventSvc.EventService
	ArticleSvc articleSvc.ArticleService
	UserSvc    userSvc.UserService
	SearchSvc  *search.SearchService // 可选：为nil时不注册semantic_search
}

// RegisterBuiltinSkills 注册所有内置Skill到Registry
func RegisterBuiltinSkills(registry *skill.Registry, deps Dependencies) {
	// 活动相关Skills
	registry.Register(event_query.New(deps.EventSvc))
	registry.Register(event_register.New(deps.EventSvc))
	registry.Register(event_cancel.New(deps.EventSvc))
	registry.Register(event_detail.New(deps.EventSvc))

	// 文章相关Skills
	registry.Register(article_query.New(deps.ArticleSvc))
	registry.Register(article_create.New(deps.ArticleSvc))
	registry.Register(article_detail.New(deps.ArticleSvc))

	// 用户相关Skills
	registry.Register(user_info.New(deps.UserSvc))
	registry.Register(user_registered_events.New(deps.EventSvc))

	// 语义检索Skill（可选，依赖Milvus）
	if deps.SearchSvc != nil {
		registry.Register(semantic_search.New(deps.SearchSvc))
	}
}
