package dto

type WxLoginRequest struct {
	Code          string `json:"code" binding:"required"`
	EncryptedData string `json:"encrypted_data" binding:"required"`
	IV            string `json:"iv" binding:"required"`
}

type BgLoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

type UserIDRequest struct {
	UserID int `uri:"id" binding:"required"`
}

// UserUpdateRequest 用户信息更新请求
type UserUpdateRequest struct {
	Nickname  *string `json:"nickname" binding:"omitempty,nickname"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url"`
	Name      *string `json:"name" binding:"omitempty,real_name"`
	Gender    *string `json:"gender" binding:"omitempty,oneof=M F U"` // M: 男, F: 女, U: 未知
	// PhoneNumber *string `json:"phone_number" binding:"omitempty,phone"`   // 手机号通过微信验证直接获取，暂时不允许更新
	Email      *string `json:"email" binding:"omitempty,email"`
	Unit       *string `json:"unit" binding:"omitempty"`
	Department *string `json:"department" binding:"omitempty"`
	Position   *string `json:"position" binding:"omitempty"`
	Industry   *string `json:"industry" binding:"omitempty"`
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	UserID       int    `json:"user_id"`
	Nickname     string `json:"nickname"`
	AvatarURL    string `json:"avatar_url"`
	Name         string `json:"name"`
	GenderCode   string `json:"gender_code"`
	Gender       string `json:"gender"`
	CountryCode  string `json:"country_code"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Unit         string `json:"unit"`
	Department   string `json:"department"`
	Position     string `json:"position"`
	Industry     string `json:"industry"`
	IndustryName string `json:"industry_name"`
	Role         string `json:"role"`
	RoleName     string `json:"role_name"`
	Status       int    `json:"status"`
}

type ListUsersRequest struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Name       string `form:"name" binding:"omitempty,max=255"`
	GenderCode string `form:"gender_code" binding:"omitempty,oneof=M F U"`
	Unit       string `form:"unit" binding:"omitempty,max=255"`
	Department string `form:"department" binding:"omitempty,max=255"`
	Position   string `form:"position" binding:"omitempty,max=255"`
	Industry   string `form:"industry" binding:"omitempty,numeric"`
	Role       string `form:"role" binding:"omitempty"`
}

type ListUsersResponse struct {
	UserID       int    `json:"user_id"`
	Nickname     string `json:"nickname"`
	AvatarURL    string `json:"avatar_url"`
	Name         string `json:"name"`
	GenderCode   string `json:"gender_code"`
	Gender       string `json:"gender"`
	CountryCode  string `json:"country_code"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Unit         string `json:"unit"`
	Department   string `json:"department"`
	Position     string `json:"position"`
	Industry     string `json:"industry"`
	IndustryName string `json:"industry_name"`
	RoleName     string `json:"role_name"`
	Status       int    `json:"status"`
	UserStatus   string `json:"user_status"`
}

type CreateAdminRequest struct {
	Nickname    string `json:"nickname" binding:"required,nickname"`
	Name        string `json:"name" binding:"omitempty,real_name"`
	AvatarURL   string `json:"avatar_url" binding:"omitempty,url"`
	PhoneNumber string `json:"phone_number" binding:"required,phone"`
	Password    string `json:"password" binding:"required"`
	Email       string `json:"email" binding:"omitempty,email"`
	Role        string `json:"role" binding:"required,oneof=ADMIN SUPERADMIN"` // ADMIN：管理员，SUPERADMIN：超级管理员
}

type UpdateAdminRequest struct {
	Nickname  *string `json:"nickname" binding:"omitempty,nickname"`
	Name      *string `json:"name" binding:"omitempty,real_name"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url"`
	Password  *string `json:"password" binding:"omitempty"`
	Email     *string `json:"email" binding:"omitempty,email"`
	Role      *string `json:"role" binding:"omitempty,oneof=ADMIN SUPERADMIN"` // ADMIN：管理员，SUPERADMIN：超级管理员
}

// UpdateAdminStatusRequest 更新管理员状态请求
type UpdateAdminStatusRequest struct {
	Operation string `json:"operation" binding:"required,oneof=ENABLE DISABLE"` // ENABLE：启用，DISABLE：禁用
}

// RefreshTokenRequest 刷新token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	// SessionSign  string `json:"session_sign" binding:"required"`
	SessionSign string `json:"session_sign" binding:"omitempty"`
}
