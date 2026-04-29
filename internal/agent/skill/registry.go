package skill

import (
	"activity-platform/internal/agent/llm"
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// SkillInfo Skill摘要信息（用于列表展示）
type SkillInfo struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	Category     string `json:"category"`
	AuthRequired bool   `json:"auth_required"`
	IsBuiltin    bool   `json:"is_builtin"`
	Description  string `json:"description"`
}

// Registry Skill注册中心
// 管理内置Skill和动态Skill的注册、查询、执行
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill // name → Skill
}

// NewRegistry 创建Skill注册中心
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register 注册一个Skill
// 如果同名Skill已存在，将覆盖
func (r *Registry) Register(s Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	meta := s.Meta()
	if _, exists := r.skills[meta.Name]; exists {
		logrus.Warnf("Skill [%s] 已存在，将被覆盖", meta.Name)
	}
	r.skills[meta.Name] = s
	logrus.Infof("注册Skill: [%s] %s (builtin=%v)", meta.Name, meta.DisplayName, meta.IsBuiltin)
}

// Unregister 注销一个Skill（仅允许注销动态Skill）
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, exists := r.skills[name]
	if !exists {
		return fmt.Errorf("Skill [%s] 不存在", name)
	}

	if s.Meta().IsBuiltin {
		return fmt.Errorf("内置Skill [%s] 不可注销", name)
	}

	delete(r.skills, name)
	logrus.Infof("注销Skill: [%s]", name)
	return nil
}

// Get 根据名称获取Skill
func (r *Registry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.skills[name]
	return s, ok
}

// GetAllToolDefinitions 获取所有已注册Skill的ToolDefinition（供LLM Function Calling使用）
func (r *Registry) GetAllToolDefinitions() []llm.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]llm.ToolDefinition, 0, len(r.skills))
	for _, s := range r.skills {
		defs = append(defs, s.Meta().ToToolDefinition())
	}
	return defs
}

// ListAllSkills 列出所有Skill的摘要信息
func (r *Registry) ListAllSkills() []SkillInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]SkillInfo, 0, len(r.skills))
	for _, s := range r.skills {
		m := s.Meta()
		list = append(list, SkillInfo{
			Name:         m.Name,
			DisplayName:  m.DisplayName,
			Category:     m.Category,
			AuthRequired: m.AuthRequired,
			IsBuiltin:    m.IsBuiltin,
			Description:  m.Description,
		})
	}
	return list
}

// Execute 执行指定名称的Skill
func (r *Registry) Execute(name string, ctx SkillContext) (SkillResult, error) {
	r.mu.RLock()
	s, ok := r.skills[name]
	r.mu.RUnlock()

	if !ok {
		return SkillResult{}, fmt.Errorf("Skill [%s] 不存在", name)
	}

	return s.Execute(ctx)
}

// ReloadDynamicSkills 从数据库重新加载动态Skill
// 由SkillService在CRUD操作后调用
func (r *Registry) ReloadDynamicSkills(ctx context.Context, dynamicSkills []Skill) {
	r.mu.Lock()
	defer r.mu.Lock()

	// 移除所有旧的动态Skill
	for name, s := range r.skills {
		if !s.Meta().IsBuiltin {
			delete(r.skills, name)
		}
	}

	// 注册新的动态Skill
	for _, s := range dynamicSkills {
		meta := s.Meta()
		r.skills[meta.Name] = s
		logrus.Infof("加载动态Skill: [%s] %s", meta.Name, meta.DisplayName)
	}
}
