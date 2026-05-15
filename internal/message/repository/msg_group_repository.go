package repository

import (
	"context"
	"errors"
	"event-platform/internal/message/dto"
	"event-platform/internal/message/model"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// MsgGroupRepository 消息群组数据访问接口
type MsgGroupRepository interface {
	// ExecTransaction 执行事务
	ExecTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	// CreateMsgGroup 创建消息群组
	CreateMsgGroup(ctx context.Context, group *model.UserMessageGroup) error
	// GetMsgGroupByID 根据ID获取消息群组
	GetMsgGroupByID(ctx context.Context, msgGroupID int) (*model.UserMessageGroup, error)
	// GetExistingMappings 查询指定群组中已存在的用户关联记录
	GetExistingMappings(ctx context.Context, groupID int, userIDs []int) (map[int]model.UserMsgGroupMapping, error)
	// CreateUserMsgGroupMappings 批量创建用户-消息群组关联记录
	CreateUserMsgGroupMappings(ctx context.Context, tx *gorm.DB, mappings []model.UserMsgGroupMapping) error
	// RecoverUserMsgGroupMappings 批量恢复用户-消息群组关联记录
	RecoverUserMsgGroupMappings(ctx context.Context, tx *gorm.DB, msgGroupID int, userIDs []int, lastReadMsgID int, operateUser int) error
	// DeleteUserMsgGroupMappings 删除用户-消息群组关联记录（软删除）
	DeleteUserMsgGroupMappings(ctx context.Context, msgGroupID int, userIDs []int, operateUser int) error
	// UpdateMsgGroup 更新消息群组信息
	UpdateMsgGroup(ctx context.Context, tx *gorm.DB, msgGroupID int, updateField map[string]interface{}) error
	// ListMsgGroups 分页查询消息群组
	ListMsgGroups(ctx context.Context, page int, pageSize int, req dto.ListMsgGroupRequest) ([]dto.ListMsgGroupResponse, int64, error)
	// GetMsgGroupByEventID 根据活动ID查询消息群组
	GetMsgGroupByEventID(ctx context.Context, eventID int) (*model.UserMessageGroup, error)
	// ListGroupsUsers 查询指定群组的用户列表
	ListGroupsUsers(ctx context.Context, page int, pageSize int, msgGroupID int) ([]dto.ListGroupsUsersResponse, int64, error)
	// ListNotInGroupUsers 查询不在指定组内的用户
	ListNotInGroupUsers(ctx context.Context, page int, pageSize int, msgGroupID int, req dto.ListNotInGroupUsersRequest) ([]dto.ListGroupsUsersResponse, int64, error)
	// GetAllUserIDs 获取所有有效用户id列表，用于全体用户入群过程
	GetAllUserIDs(ctx context.Context, page int) ([]int, error)
	// GetAllUserGroupIDs 获取所有包含全体用户的群组ID
	GetAllUserGroupIDs(ctx context.Context) ([]int, error)
	// DeleteUserByGroupID 删除指定群组内的全部用户
	DeleteUserByGroupID(ctx context.Context, tx *gorm.DB, msgGroupID int, updateField map[string]interface{}) error
}

// MsgGroupRepositoryImpl 实现消息群组数据访问接口的具体结构体
type MsgGroupRepositoryImpl struct {
	db          *gorm.DB
	messageRepo MessageRepository
}

// NewMsgGroupRepository 创建消息群组数据访问实例
func NewMsgGroupRepository(db *gorm.DB, messageRepo MessageRepository) MsgGroupRepository {
	return &MsgGroupRepositoryImpl{db: db, messageRepo: messageRepo}
}

// ExecTransaction 实现事务执行（使用 GORM 的 Transaction 方法）
func (repo *MsgGroupRepositoryImpl) ExecTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return repo.db.WithContext(ctx).Transaction(fn)
}

// CreateMsgGroup 创建消息群组
func (repo *MsgGroupRepositoryImpl) CreateMsgGroup(ctx context.Context, group *model.UserMessageGroup) error {
	err := repo.db.WithContext(ctx).Create(group).Error
	if err != nil {
		exist, _, _ := utils.IsUniqueConstraintError(err)
		if exist {
			return utils.NewBusinessError(utils.ErrCodeResourceExists, "已存在同名消息群组")
		}
		return utils.NewSystemError(fmt.Errorf("创建消息群组失败: %v", err))
	}
	return nil
}

// GetMsgGroupByID 根据ID获取消息群组
func (repo *MsgGroupRepositoryImpl) GetMsgGroupByID(ctx context.Context, msgGroupID int) (*model.UserMessageGroup, error) {
	var group model.UserMessageGroup
	err := repo.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", msgGroupID, utils.DeletedFlagNo).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询消息群组失败: %v", err))
	}
	return &group, nil
}

// GetExistingMappings 查询指定群组中已存在的用户关联记录
func (repo *MsgGroupRepositoryImpl) GetExistingMappings(ctx context.Context, groupID int, userIDs []int) (map[int]model.UserMsgGroupMapping, error) {
	var mappings []model.UserMsgGroupMapping
	if err := repo.db.WithContext(ctx).
		Where("msg_group_id = ? AND user_id IN (?)", groupID, userIDs).
		Find(&mappings).Error; err != nil {
		return nil, err
	}

	// 转换为 map[userID]mapping，便于快速查询
	result := make(map[int]model.UserMsgGroupMapping, len(mappings))
	for _, m := range mappings {
		result[m.UserID] = m
	}
	return result, nil
}

// CreateUserMsgGroupMappings 批量创建用户-消息群组关联记录
func (repo *MsgGroupRepositoryImpl) CreateUserMsgGroupMappings(ctx context.Context, tx *gorm.DB, mappings []model.UserMsgGroupMapping) error {
	if len(mappings) == 0 {
		return nil
	}
	if err := tx.Create(&mappings).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("批量创建用户-消息群组关联记录失败: %v", err))
	}
	return nil
}

// RecoverUserMsgGroupMappings 批量恢复用户-消息群组关联记录
func (repo *MsgGroupRepositoryImpl) RecoverUserMsgGroupMappings(ctx context.Context, tx *gorm.DB, msgGroupID int, userIDs []int, lastReadMsgID int, operateUser int) error {
	if len(userIDs) == 0 {
		return nil
	}

	if err := tx.Model(&model.UserMsgGroupMapping{}).
		Where("msg_group_id = ? AND user_id in (?)", msgGroupID, userIDs).
		Updates(map[string]interface{}{
			"is_deleted":       utils.DeletedFlagNo,
			"last_read_msg_id": lastReadMsgID,
			"join_msg_id":      lastReadMsgID,
			"update_user":      operateUser,
		}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("批量恢复用户-消息群组关联记录失败: %v", err))
	}

	return nil
}

// DeleteUserMsgGroupMappings 删除用户-消息群组关联记录（软删除）
func (repo *MsgGroupRepositoryImpl) DeleteUserMsgGroupMappings(ctx context.Context, msgGroupID int, userIDs []int, operateUser int) error {
	if len(userIDs) == 0 {
		return nil
	}

	if err := repo.db.WithContext(ctx).Model(&model.UserMsgGroupMapping{}).
		Where("msg_group_id = ? AND user_id in (?) AND is_deleted = ?", msgGroupID, userIDs, utils.DeletedFlagNo).
		Updates(map[string]interface{}{
			"is_deleted":  utils.DeletedFlagYes,
			"update_user": operateUser,
		}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("批量删除用户-消息群组关联记录失败: %v", err))
	}

	return nil
}

// UpdateMsgGroup 更新消息群组信息
func (repo *MsgGroupRepositoryImpl) UpdateMsgGroup(ctx context.Context, tx *gorm.DB, msgGroupID int, updateField map[string]interface{}) error {
	var err error
	if tx == nil {
		err = repo.db.WithContext(ctx).Model(&model.UserMessageGroup{}).
			Where("id = ?", msgGroupID).
			Updates(updateField).Error
	} else {
		err = tx.Model(&model.UserMessageGroup{}).
			Where("id = ?", msgGroupID).
			Updates(updateField).Error
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return utils.NewSystemError(fmt.Errorf("更新消息群组失败: %v", err))
	}
	return nil
}

// ListMsgGroups 分页查询消息群组
func (repo *MsgGroupRepositoryImpl) ListMsgGroups(ctx context.Context, page int, pageSize int, req dto.ListMsgGroupRequest) ([]dto.ListMsgGroupResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	var groups []dto.ListMsgGroupResponse

	query := repo.db.WithContext(ctx).Table("user_message_groups umg").
		Select("umg.id, umg.group_name, umg.desc, umg.event_id, e.title AS event_title, umg.include_all_user, umg.is_deleted, COALESCE(member_counts.count, 0) AS member_count").
		Joins("LEFT JOIN events e ON e.id = umg.event_id").
		Joins(`
			LEFT JOIN (
				SELECT msg_group_id, COUNT(*) AS count 
				FROM user_msg_group_mappings 
				WHERE is_deleted = ?
				GROUP BY msg_group_id 
				) member_counts ON member_counts.msg_group_id = umg.id`, utils.DeletedFlagNo)
	// 拼接查询条件
	if req.GroupName != "" {
		query = query.Where("umg.group_name LIKE ?", "%"+req.GroupName+"%")
	}
	if req.EventID != 0 {
		query = query.Where("umg.event_id = ?", req.EventID)
	}
	if req.EventTitle != "" {
		query = query.Where("e.title LIKE ?", "%"+req.EventTitle+"%")
	}
	if req.QueryScope != "" {
		// 如果传入了查询范围，则添加查询条件
		// 如果传入了查询范围为DELETED，则查询已删除的群组
		if req.QueryScope == utils.QueryScopeDeleted {
			query = query.Where("umg.is_deleted = ?", utils.DeletedFlagYes) // 查询已删除的文章
		}
		if req.QueryScope == utils.QueryScopeAll {
			// 如果传入了查询范围为ALL，则查询所有群组，包括已删除和未删除的
		}
	} else {
		// 默认查询未删除群组
		query = query.Where("umg.is_deleted = ?", utils.DeletedFlagNo)
	}

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return groups, total, nil
}

// GetMsgGroupByEventID 根据活动ID查询消息群组
func (repo *MsgGroupRepositoryImpl) GetMsgGroupByEventID(ctx context.Context, eventID int) (*model.UserMessageGroup, error) {
	var msgGroup model.UserMessageGroup
	if err := repo.db.WithContext(ctx).
		Where("event_id = ? AND is_deleted = ?", eventID, utils.DeletedFlagNo).
		First(&msgGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询消息群组失败: %v", err))
	}
	return &msgGroup, nil
}

// ListGroupsUsers 查询指定群组的用户列表
func (repo *MsgGroupRepositoryImpl) ListGroupsUsers(ctx context.Context, page int, pageSize int, msgGroupID int) ([]dto.ListGroupsUsersResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	var users []dto.ListGroupsUsersResponse
	query := repo.db.WithContext(ctx)

	query = query.Table("users u").
		Select(`
				u.user_id, u.nickname, u.name, u.gender AS gender_code, 
				CASE 
					WHEN u.gender = 'M' THEN '男' 
					WHEN gender = 'F' THEN '女' 
					ELSE '未知'
				END AS gender,
				u.phone_number, u.email, u.unit, u.department, u.position, 
				u.industry, i.industry_name`).
		Joins("LEFT JOIN industries i ON u.industry = i.industry_code").
		Joins("JOIN user_msg_group_mappings m ON u.user_id = m.user_id").
		Where("m.msg_group_id = ? AND m.is_deleted = ?", msgGroupID, utils.DeletedFlagNo)

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}
	return users, total, nil
}

// ListNotInGroupUsers 查询不在指定组内的用户
func (repo *MsgGroupRepositoryImpl) ListNotInGroupUsers(ctx context.Context, page int, pageSize int, msgGroupID int, req dto.ListNotInGroupUsersRequest) ([]dto.ListGroupsUsersResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	var users []dto.ListGroupsUsersResponse
	query := repo.db.WithContext(ctx)

	query = query.Table("users u").
		Select(`u.user_id, u.nickname, u.name, u.gender AS gender_code, 
				CASE 
					WHEN u.gender = 'M' THEN '男' 
					WHEN gender = 'F' THEN '女' 
					ELSE '未知'
				END AS gender,
				u.phone_number, u.email, u.unit, u.department, u.position, 
				u.industry, i.industry_name`).
		Joins("LEFT JOIN industries i ON u.industry = i.industry_code").
		Where(`NOT EXISTS (
						SELECT 1  
						FROM user_msg_group_mappings m  
						WHERE m.user_id = u.user_id 
						AND m.msg_group_id = ?
						AND m.is_deleted = ?)`, msgGroupID, utils.DeletedFlagNo)

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

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}
	return users, total, nil
}

// GetAllUserIDs 获取所有有效用户id列表，用于全体用户入群过程
func (repo *MsgGroupRepositoryImpl) GetAllUserIDs(ctx context.Context, page int) ([]int, error) {
	pageSize := 200
	offset := (page - 1) * pageSize

	var userIDs []int
	err := repo.db.WithContext(ctx).Table("users").Limit(pageSize).Offset(offset).Pluck("user_id", &userIDs).Error

	return userIDs, err
}

// GetAllUserGroupIDs 获取所有包含全体用户的群组ID
func (repo *MsgGroupRepositoryImpl) GetAllUserGroupIDs(ctx context.Context) ([]int, error) {
	var groupIDs []int
	err := repo.db.WithContext(ctx).Table("user_message_groups").Where("include_all_user = ?", utils.FlagYes).Pluck("id", &groupIDs).Error

	return groupIDs, err
}

// DeleteUserByGroupID 删除指定群组内的全部用户
func (repo *MsgGroupRepositoryImpl) DeleteUserByGroupID(ctx context.Context, tx *gorm.DB, msgGroupID int, updateField map[string]interface{}) error {
	var err error
	if tx == nil {
		err = repo.db.WithContext(ctx).Model(&model.UserMsgGroupMapping{}).
			Where("msg_group_id = ?", msgGroupID).
			Updates(updateField).Error
	} else {
		err = tx.Model(&model.UserMsgGroupMapping{}).
			Where("msg_group_id = ?", msgGroupID).
			Updates(updateField).Error
	}
	if err != nil {
		return utils.NewSystemError(fmt.Errorf("删除群组[%d]内用户失败: %v", msgGroupID, err))
	}
	return nil
}
