package model

import (
	"time"
)

// User 数据模型
type User struct {
	UserID        int       `json:"user_id" gorm:"primaryKey;column:user_id"`
	OpenID        string    `json:"openid" gorm:"column:openid;default:NULL"`
	UnionID       string    `json:"unionid" gorm:"column:unionid;default:NULL"`
	SessionKey    string    `json:"session_key" gorm:"column:session_key"`
	Nickname      string    `json:"nickname" gorm:"column:nickname"`
	AvatarURL     string    `json:"avatar_url" gorm:"column:avatar_url"`
	Name          string    `json:"name" gorm:"column:name"`
	Gender        string    `json:"gender" gorm:"column:gender"` // M: 男, F: 女, U: 未知
	CountryCode   string    `json:"country_code" gorm:"column:country_code"`
	PhoneNumber   string    `json:"phone_number" gorm:"column:phone_number;default:NULL"`
	Email         string    `json:"email" gorm:"column:email"`
	Region        string    `json:"region" gorm:"column:region"`
	Status        int       `json:"status" gorm:"column:status;default:1"` // 默认=1，1：正常，2：禁用
	LastLoginTime time.Time `json:"last_login_time" gorm:"column:last_login_time;autoUpdateTime"`
	UserLevel     int       `json:"user_level" gorm:"column:user_level"`
	Password      string    `json:"password" gorm:"column:password"`
	Role          string    `json:"role" gorm:"column:role;default:'USER'"` // 默认=USER
	Unit          string    `json:"unit" gorm:"column:unit;default:NULL"`
	Department    string    `json:"department" gorm:"column:department;default:NULL"`
	Position      string    `json:"position" gorm:"column:position;default:NULL"`
	Industry      string    `json:"industry" gorm:"column:industry;default:NULL"`
	RefreshToken  string    `json:"refresh_token" gorm:"column:refresh_token"` // 刷新token
	CreateTime    time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime    time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
	CreateUser    int       `json:"create_user" gorm:"column:create_user"` // 创建人ID
	UpdateUser    int       `json:"update_user" gorm:"column:update_user"` // 最后更新人ID
}

// TableName 设置表名
func (*User) TableName() string {
	return "users"
}
