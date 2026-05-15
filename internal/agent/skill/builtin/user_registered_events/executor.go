package user_registered_events

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"event-platform/internal/agent/skill"
	eventSvc "event-platform/internal/event/service"
)

//go:embed skill.md
var skillMD string

type userRegisteredEventsSkill struct {
	meta     skill.SkillMeta
	eventSvc eventSvc.EventService
}

func New(eventSvc eventSvc.EventService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析user_registered_events/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &userRegisteredEventsSkill{
		meta:     meta,
		eventSvc: eventSvc,
	}
}

func (s *userRegisteredEventsSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *userRegisteredEventsSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	if ctx.UserID == 0 {
		return skill.SkillResult{
			Success: false,
			Message: "请先登录后查询已报名活动",
		}, nil
	}

	page := skill.GetParamIntWithDefault(ctx.Params, "page", 1)
	pageSize := skill.GetParamIntWithDefault(ctx.Params, "page_size", 10)
	eventStatus, _ := skill.GetParamString(ctx.Params, "event_status")

	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}

	events, total, err := s.eventSvc.ListUserRegisteredEvents(ctx.Ctx, page, pageSize, ctx.UserID, eventStatus)
	if err != nil {
		return skill.SkillResult{}, fmt.Errorf("查询已报名活动失败: %w", err)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"events":    events,
	})

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: fmt.Sprintf("您已报名 %d 个活动，共 %d 个", len(events), total),
	}, nil
}
