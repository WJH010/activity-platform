package utils

// 角色权限映射表: 键为角色，值为该角色可访问的所有角色权限集合
var roleAccessMap = map[string]map[string]bool{
	RoleUser: {
		RoleUser: true, // 普通用户只能访问自己权限的接口
	},
	RoleAdmin: {
		RoleUser:  true, // 管理员可以访问普通用户权限的接口
		RoleAdmin: true, // 管理员可以访问管理员权限的接口
	},
	RoleSuperAdmin: {
		RoleUser:       true, // 超级管理员可以访问普通用户权限的接口
		RoleAdmin:      true, // 超级管理员可以访问管理员权限的接口
		RoleSuperAdmin: true, // 超级管理员可以访问超级管理员权限的接口
	},
}

// 检查用户角色是否有权限访问目标角色权限的接口
func HasAccess(userRole, targetRole string) bool {
	// 获取该用户角色可访问的权限集合
	accessibleRoles, exists := roleAccessMap[userRole]
	if !exists {
		return false
	}
	// 检查是否包含目标权限
	return accessibleRoles[targetRole]
}
