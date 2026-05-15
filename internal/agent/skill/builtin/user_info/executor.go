package user_info

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"event-platform/internal/agent/skill"
	userSvc "event-platform/internal/user/service"
)

//go:embed skill.md
var skillMD string

type userInfoSkill struct {
	meta    skill.SkillMeta
	userSvc userSvc.UserService
}

func New(userSvc userSvc.UserService) skill.Skill {
	meta, err := skill.ParseSkillMD(skillMD)
	if err != nil {
		panic(fmt.Sprintf("解析user_info/skill.md失败: %v", err))
	}
	meta.IsBuiltin = true

	return &userInfoSkill{
		meta:    meta,
		userSvc: userSvc,
	}
}

func (s *userInfoSkill) Meta() skill.SkillMeta {
	return s.meta
}

func (s *userInfoSkill) Execute(ctx skill.SkillContext) (skill.SkillResult, error) {
	if ctx.UserID == 0 {
		return skill.SkillResult{
			Success: false,
			Message: "请先登录后查询个人信息",
		}, nil
	}

	user, err := s.userSvc.GetUserByID(ctx.Ctx, ctx.UserID)
	if err != nil {
		return skill.SkillResult{
			Success: false,
			Message: fmt.Sprintf("查询用户信息失败: %s", err.Error()),
		}, nil
	}

	data, _ := json.Marshal(user)

	return skill.SkillResult{
		Success: true,
		Data:    json.RawMessage(data),
		Message: "用户信息获取成功",
	}, nil
}
