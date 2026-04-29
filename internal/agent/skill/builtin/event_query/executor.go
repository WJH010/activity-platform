package event_query

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"activity-platform/internal/agent/skill"
	eventSvc "activity-platform/internal/event/service"
)

//go:embed skill.md
var skillMD string

type eventQuerySkill struct {
	meta     skill.SkillMeta
	eventSvc eventSvc.EventService
}

func New(eventSvc eventSvc.EventService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析event_query/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &eventQuerySkill{
		meta:     meta,
		eventSvc: eventSvc,
	}
}

func (s *eventQuerySkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *eventQuerySkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	page := skill.GetParamIntWithDefault(ctx.Params, "page", 1)
	pageSize := skill.GetParamIntWithDefault(ctx.Params, "page_size", 10)
	eventStatus, _ := skill.GetParamString(ctx.Params, "event_status")
	eventTitle, _ := skill.GetParamString(ctx.Params, "event_title")

	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}

	// 调用EventService
	events, total, err := s.eventSvc.ListEvent(ctx.Ctx, page, pageSize, eventStatus, "", eventTitle)
	if err != nil {
		return skill.SkillResult{}, fmt.Errorf("查询活动列表失败: %w", err)
	}

	// 构建结果
	data, _ := json.Marshal(map[string]interface{}{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"events":    events,
	})

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: fmt.Sprintf("查询到 %d 条活动，共 %d 条", len(events), total),
	}, nil
}
