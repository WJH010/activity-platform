package event_detail

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	_ "embed"

	"activity-platform/internal/agent/skill"
	eventSvc "activity-platform/internal/event/service"
)

//go:embed skill.md
var skillMD string

// maxEventDetailChars Detail字段返回给LLM的最大字符数
const maxEventDetailChars = 2000

type eventDetailSkill struct {
	meta     skill.SkillMeta
	eventSvc eventSvc.EventService
}

func New(eventSvc eventSvc.EventService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析event_detail/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &eventDetailSkill{
		meta:     meta,
		eventSvc: eventSvc,
	}
}

func (s *eventDetailSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *eventDetailSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	eventID, ok := skill.GetParamInt(ctx.Params, "event_id")
	if !ok {
		return skill.SkillResult{
			Success: false,
			Message: "缺少必要参数: event_id",
		}, nil
	}

	event, err := s.eventSvc.GetEventDetail(ctx.Ctx, eventID)
	if err != nil {
		return skill.SkillResult{
			Success: false,
			Message: fmt.Sprintf("查询活动详情失败: %s", err.Error()),
		}, nil
	}

	// 截断 Detail 字段（HTML内容可能非常长）
	truncatedDetail := truncateRunes(event.Detail, maxEventDetailChars)
	if utf8.RuneCountInString(event.Detail) > maxEventDetailChars {
		truncatedDetail += "\n...[详情过长已截断，如需完整内容请告知用户前往活动详情页查看]"
	}
	event.Detail = truncatedDetail

	data, _ := json.Marshal(event)

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: fmt.Sprintf("活动【%s】详情获取成功", event.Title),
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
