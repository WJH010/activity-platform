package event_cancel

import (
	"fmt"

	_ "embed"

	"event-platform/internal/agent/skill"
	eventSvc "event-platform/internal/event/service"
)

//go:embed skill.md
var skillMD string

type eventCancelSkill struct {
	meta     skill.SkillMeta
	eventSvc eventSvc.EventService
}

func New(eventSvc eventSvc.EventService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析event_cancel/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &eventCancelSkill{
		meta:     meta,
		eventSvc: eventSvc,
	}
}

func (s *eventCancelSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *eventCancelSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	eventID, ok := skill.GetParamInt(ctx.Params, "event_id")
	if !ok {
		return skill.SkillResult{
			Success: false,
			Message: "缺少必要参数: event_id",
		}, nil
	}

	if ctx.UserID == 0 {
		return skill.SkillResult{
			Success: false,
			Message: "请先登录后再操作",
		}, nil
	}

	err := s.eventSvc.CancelRegistrationEvent(ctx.Ctx, eventID, ctx.UserID)
	if err != nil {
		return skill.SkillResult{
			Success: false,
			Message: fmt.Sprintf("取消报名失败: %s", err.Error()),
		}, nil
	}

	return skill.SkillResult{
		Success: true,
		Data: map[string]interface{}{
			"event_id": eventID,
		},
		Message: fmt.Sprintf("已成功取消活动(ID:%d)的报名", eventID),
	}, nil
}
