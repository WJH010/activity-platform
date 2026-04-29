package event_register

import (
	"fmt"

	_ "embed"

	"activity-platform/internal/agent/skill"
	eventSvc "activity-platform/internal/event/service"
)

//go:embed skill.md
var skillMD string

type eventRegisterSkill struct {
	meta     skill.SkillMeta
	eventSvc eventSvc.EventService
}

func New(eventSvc eventSvc.EventService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析event_register/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &eventRegisterSkill{
		meta:     meta,
		eventSvc: eventSvc,
	}
}

func (s *eventRegisterSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *eventRegisterSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
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
			Message: "请先登录后再报名活动",
		}, nil
	}

	err := s.eventSvc.RegistrationEvent(ctx.Ctx, eventID, ctx.UserID)
	if err != nil {
		return skill.SkillResult{
			Success: false,
			Message: fmt.Sprintf("报名失败: %s", err.Error()),
		}, nil
	}

	return skill.SkillResult{
		Success: true,
		Data: map[string]interface{}{
			"event_id": eventID,
		},
		Message: fmt.Sprintf("报名成功！活动ID: %d", eventID),
	}, nil
}
