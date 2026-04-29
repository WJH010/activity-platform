package repository

import (
	"activity-platform/internal/user/dto"
	"activity-platform/internal/user/model"
	"activity-platform/internal/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	GetUserByOpenID(ctx context.Context, openid string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, userID int, updateFields map[string]any) error
	UpdateSessionAndLoginTime(ctx context.Context, userID int, sessionKey string, countryCode string, phoneNumber string) error
	GetUserInfoByID(ctx context.Context, userID int) (*dto.UserInfoResponse, error)
	ListAllUsers(ctx context.Context, page, pageSize int, req dto.ListUsersRequest) ([]*dto.ListUsersResponse, int64, error)
	// GetPasswordByPhone 根据手机号获取密码
	GetPasswordByPhone(ctx context.Context, phoneNumber string) (*model.User, error)
	// UpdateRefreshToken 更新用户刷新token
	UpdateRefreshToken(ctx context.Context, userID int, refreshToken string) error
	// GetUserByID 根据用户ID获取所有用户信息
	GetUserByID(ctx context.Context, userID int) (*model.User, error)
}

// UserRepositoryImpl 用户仓库实现
type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

// GetUserByOpenID 根据 openid 获取用户信息
func (repo *UserRepositoryImpl) GetUserByOpenID(ctx context.Context, openid string) (*model.User, error) {
	var user model.User
	result := repo.db.WithContext(ctx).Where("openid = ?", openid).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", result.Error))
	}
	return &user, nil
}

// Create 创建新用户
func (repo *UserRepositoryImpl) Create(ctx context.Context, user *model.User) error {
	err := repo.db.WithContext(ctx).Create(user).Error
	if err != nil {
		exist, fieldName, _ := utils.IsUniqueConstraintError(err)
		if exist && fieldName == "phone_number" {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "手机号已被注册")
		}
		return utils.NewSystemError(fmt.Errorf("创建用户失败: %w", err))
	}
	return err
}

// UpdateSessionAndLoginTime 登录时更新session_key和最后登录时间
func (repo *UserRepositoryImpl) UpdateSessionAndLoginTime(ctx context.Context, userID int, sessionKey string, countryCode string, phoneNumber string) error {
	result := repo.db.WithContext(ctx).Model(&model.User{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"session_key":     sessionKey,
			"country_code":    countryCode,
			"phone_number":    phoneNumber,
			"last_login_time": time.Now(),
		})

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新登录信息失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewSystemError(fmt.Errorf("更新登录信息异常，未更新任何数据: %w", result.Error))
	}

	return nil
}

// GetUserInfoByID 获取用户信息
func (repo *UserRepositoryImpl) GetUserInfoByID(ctx context.Context, userID int) (*dto.UserInfoResponse, error) {
	var user dto.UserInfoResponse
	query := repo.db.WithContext(ctx)

	result := query.Table("users u").
		Select(`u.user_id, u.nickname, u.avatar_url, u.name, u.gender AS gender_code,
				CASE
					WHEN gender = 'M' THEN
					'男'
					WHEN gender = 'F' THEN
					'女'
					ELSE
					'未知'
				END AS gender,u.country_code, u.phone_number, u.email, u.unit, u.department, u.position, u.industry, i.industry_name, u.role, ur.role_name, u.status`).
		Joins("LEFT JOIN industries i ON u.industry = i.industry_code").
		Joins("LEFT JOIN user_role ur ON ur.role_code = u.role").
		Where("user_id = ?", userID).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询用户信息失败: %v", result.Error))
	}
	return &user, nil
}

// Update 更新用户信息
func (repo *UserRepositoryImpl) Update(ctx context.Context, userID int, updateFields map[string]any) error {

	result := repo.db.WithContext(ctx).Model(&model.User{}).
		Where("user_id = ?", userID).
		Updates(updateFields)

	if result.Error != nil {
		return utils.NewSystemError(fmt.Errorf("更新用户信息失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "更新用户信息失败，用户数据异常，请刷新页面后重试")
	}
	return nil
}

// ListAllUsers 分页查询用户列表
func (repo *UserRepositoryImpl) ListAllUsers(ctx context.Context, page, pageSize int, req dto.ListUsersRequest) ([]*dto.ListUsersResponse, int64, error) {
	var users []*dto.ListUsersResponse
	var total int64

	query := repo.db.WithContext(ctx).Table("users u").
		Select(`u.user_id, u.nickname, u.avatar_url, u.name, u.gender AS gender_code,
				CASE
					WHEN gender = 'M' THEN
					'男'
					WHEN gender = 'F' THEN
					'女'
					ELSE
					'未知'
				END AS gender,
				u.country_code, u.phone_number, u.email, u.unit, u.department, u.position, u.industry, i.industry_name, ur.role_name,
				u.status,
				CASE
					u.status
					WHEN 1 THEN
					"已启用"
					WHEN 2 THEN
					"已禁用"
				END AS user_status
				`).
		Joins("LEFT JOIN industries i ON u.industry = i.industry_code").
		Joins("LEFT JOIN user_role ur ON ur.role_code = u.role")

	// 拼接查询条件
	if req.Name != "" {
		query = query.Where("u.name LIKE ?", "%"+req.Name+"%")
	}
	if req.GenderCode != "" {
		query = query.Where("u.gender = ?", req.GenderCode)
	}
	if req.Unit != "" {
		query = query.Where("u.unit LIKE ?", "%"+req.Unit+"%")
	}
	if req.Department != "" {
		query = query.Where("u.department LIKE ?", "%"+req.Department+"%")
	}
	if req.Position != "" {
		query = query.Where("u.position LIKE ?", "%"+req.Position+"%")
	}
	if req.Industry != "" {
		query = query.Where("u.industry = ?", req.Industry)
	}
	if req.Role != "" {
		query = query.Where("u.role = ?", req.Role)
	}

	// 计算总记录数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算用户总数失败: %w", err))
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("查询用户列表失败: %w", err))
	}

	return users, total, nil
}

// GetPasswordByPhone 根据手机号获取密码
func (repo *UserRepositoryImpl) GetPasswordByPhone(ctx context.Context, phoneNumber string) (*model.User, error) {
	var user *model.User
	if err := repo.db.WithContext(ctx).Where("phone_number = ? AND status = ?", phoneNumber, 1).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "用户不存在或账号已被禁用")
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询用户失败: %w", err))
	}
	return user, nil
}

// UpdateRefreshToken 更新用户刷新token
func (repo *UserRepositoryImpl) UpdateRefreshToken(ctx context.Context, userID int, refreshToken string) error {
	if err := repo.db.WithContext(ctx).Model(&model.User{}).
		Where("user_id = ?", userID).
		Update("refresh_token", refreshToken).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("更新刷新token失败: %w", err))
	}
	return nil
}

// GetUserByID 根据用户ID获取所有用户信息
func (repo *UserRepositoryImpl) GetUserByID(ctx context.Context, userID int) (*model.User, error) {
	var user model.User
	if err := repo.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, 1).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "用户不存在或账号已被禁用")
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询用户失败: %w", err))
	}
	return &user, nil
}
