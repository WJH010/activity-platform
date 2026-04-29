package model

// UserRole 对应 news_platform 数据库中 user_role 数据表的数据模型
type UserRole struct {
	ID       int    `json:"id" gorm:"primaryKey;column:id"`                      // 主键ID
	RoleCode string `json:"role_code" gorm:"not null;size:50;column:role_code"`  // 角色编码
	RoleName string `json:"role_name" gorm:"not null;size:255;column:role_name"` // 角色名称
}

// TableName 设置当前模型对应的数据库表名
func (*UserRole) TableName() string {
	return "user_role"
}
