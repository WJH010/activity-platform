package dto

// CreateSkillRequest 创建Skill请求
type CreateSkillRequest struct {
	Name            string `json:"name" binding:"required,max=100"`
	DisplayName     string `json:"display_name" binding:"required,max=200"`
	Description     string `json:"description" binding:"required"`
	ParamSchema     string `json:"param_schema"`
	HttpMethod      string `json:"http_method" binding:"required,oneof=GET POST PUT DELETE"`
	UrlTemplate     string `json:"url_template" binding:"required"`
	HeadersTemplate string `json:"headers_template"`
	BodyTemplate    string `json:"body_template"`
	AuthRequired    *int   `json:"auth_required"`
	Category        string `json:"category" binding:"max=50"`
	SortOrder       *int   `json:"sort_order"`
}

// UpdateSkillRequest 更新Skill请求
type UpdateSkillRequest struct {
	DisplayName     *string `json:"display_name" binding:"omitempty,non_empty_string,max=200"`
	Description     *string `json:"description" binding:"omitempty"`
	ParamSchema     *string `json:"param_schema"`
	HttpMethod      *string `json:"http_method" binding:"omitempty,oneof=GET POST PUT DELETE"`
	UrlTemplate     *string `json:"url_template" binding:"omitempty,non_empty_string"`
	HeadersTemplate *string `json:"headers_template"`
	BodyTemplate    *string `json:"body_template"`
	AuthRequired    *int    `json:"auth_required" binding:"omitempty,oneof=0 1"`
	IsEnabled       *int    `json:"is_enabled" binding:"omitempty,oneof=0 1"`
	Category        *string `json:"category" binding:"omitempty,max=50"`
	SortOrder       *int    `json:"sort_order"`
}

// SkillResponse Skill响应
type SkillResponse struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	ParamSchema     string `json:"param_schema"`
	HttpMethod      string `json:"http_method"`
	UrlTemplate     string `json:"url_template"`
	HeadersTemplate string `json:"headers_template"`
	BodyTemplate    string `json:"body_template"`
	AuthRequired    int    `json:"auth_required"`
	IsEnabled       int    `json:"is_enabled"`
	IsBuiltin       int    `json:"is_builtin"`
	Category        string `json:"category"`
	SortOrder       int    `json:"sort_order"`
}

// ToggleSkillRequest 启用/禁用Skill请求
type ToggleSkillRequest struct {
	IsEnabled int `json:"is_enabled" binding:"required,oneof=0 1"`
}
