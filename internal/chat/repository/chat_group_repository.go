package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"activity-platform/internal/chat/dto"
	"activity-platform/internal/chat/model"
	"activity-platform/internal/utils"
)

// ChatGroupRepository 定义群组数据访问接口
type ChatGroupRepository interface {
	CreateGroup(ctx context.Context, tx *gorm.DB, group *model.ChatGroup) error
	CreateMember(ctx context.Context, tx *gorm.DB, member *model.ChatGroupMember) error
	GetGroupByID(ctx context.Context, groupID int) (*model.ChatGroup, error)
	GetMemberUserIDs(ctx context.Context, groupID int) ([]int, error)
	AddMembers(ctx context.Context, tx *gorm.DB, members []*model.ChatGroupMember) error
	RemoveMembers(ctx context.Context, tx *gorm.DB, groupID int, userIDs []int) error
	ListGroupMembers(ctx context.Context, groupID int, page, pageSize int) ([]dto.ListGroupMembersResponse, int64, error)
	UpdateLatestMessageID(ctx context.Context, tx *gorm.DB, groupID int, messageID int64) error
	UpdateMemberLastReadMsgID(ctx context.Context, groupID int, userID int, messageID int64) error
	HasUnreadMessages(ctx context.Context, userID int) (bool, error)
	ListUserGroups(ctx context.Context, userID int) ([]dto.ListUserGroupsResponse, error)
	GetMember(ctx context.Context, groupID int, userID int) (*model.ChatGroupMember, error)
	ListMemberIDsByGroupID(ctx context.Context, groupID int) ([]int, error)
	ListGroupMessages(ctx context.Context, groupID int, joinMsgID int64, req dto.ListGroupMessagesReq) ([]dto.ListGroupMessagesResponse, int64, error)
	DeleteGroup(ctx context.Context, tx *gorm.DB, groupID int) error
	AddUserToGroup(ctx context.Context, groupID int, userID int) error
	DeleteUserFromGroup(ctx context.Context, groupID int, userID int) error
	ListAllGroups(ctx context.Context, page, pageSize int) ([]model.ChatGroup, int64, error)
	FindUsersNotInGroup(ctx context.Context, groupID int, page, pageSize int, name string) ([]dto.NotInGroupUserResponse, int64, error)
}

// chatGroupRepositoryImpl 实现接口
type chatGroupRepositoryImpl struct {
	db *gorm.DB
}

// NewChatGroupRepository 创建实例
func NewChatGroupRepository(db *gorm.DB) ChatGroupRepository {
	return &chatGroupRepositoryImpl{db: db}
}

// CreateGroup 创建一个新的群组
func (repo *chatGroupRepositoryImpl) CreateGroup(ctx context.Context, tx *gorm.DB, group *model.ChatGroup) error {
	if err := tx.WithContext(ctx).Create(group).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建群组失败: %w", err))
	}
	return nil
}

// RemoveMembers 批量移除群组成员
func (repo *chatGroupRepositoryImpl) RemoveMembers(ctx context.Context, tx *gorm.DB, groupID int, userIDs []int) error {
	if err := tx.WithContext(ctx).
		Where("group_id = ? AND user_id IN ?", groupID, userIDs).
		Delete(&model.ChatGroupMember{}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("移除群组成员失败: %w", err))
	}
	return nil
}

// ListGroupMembers 分页查询群组成员的详细信息
func (repo *chatGroupRepositoryImpl) ListGroupMembers(ctx context.Context, groupID int, page, pageSize int) ([]dto.ListGroupMembersResponse, int64, error) {
	var users []dto.ListGroupMembersResponse
	var total int64

	// 构建基础查询
	query := repo.db.WithContext(ctx).Table("users u").
		Select("u.user_id, u.nickname, u.name, u.avatar_url").
		Joins("JOIN chat_group_members cgm ON u.user_id = cgm.user_id").
		Where("cgm.group_id = ?", groupID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("查询群成员总数失败: %w", err))
	}

	// 应用分页并查询数据
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("分页查询群成员列表失败: %w", err))
	}

	return users, total, nil
}

// UpdateLatestMessageID 更新群组的最新消息ID
func (repo *chatGroupRepositoryImpl) UpdateLatestMessageID(ctx context.Context, tx *gorm.DB, groupID int, messageID int64) error {
	if err := tx.WithContext(ctx).Model(&model.ChatGroup{}).Where("id = ?", groupID).Update("latest_message_id", messageID).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("更新群组最新消息ID失败: %w", err))
	}
	return nil
}

// CreateMember 创建一个新的群组成员
func (repo *chatGroupRepositoryImpl) CreateMember(ctx context.Context, tx *gorm.DB, member *model.ChatGroupMember) error {
	member.JoinedAt = time.Now()
	if err := tx.WithContext(ctx).Create(member).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建群组成员失败: %w", err))
	}
	return nil
}

// AddMembers 批量添加群组成员
func (repo *chatGroupRepositoryImpl) AddMembers(ctx context.Context, tx *gorm.DB, members []*model.ChatGroupMember) error {
	if len(members) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Create(&members).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("批量添加群组成员失败: %w", err))
	}
	return nil
}

// GetGroupByID 根据 ID 查询群组
func (repo *chatGroupRepositoryImpl) GetGroupByID(ctx context.Context, groupID int) (*model.ChatGroup, error) {
	var group model.ChatGroup
	if err := repo.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 未找到，返回 nil, nil
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询群组失败: %w", err))
	}
	return &group, nil
}

// GetMemberUserIDs 获取群组的所有成员 UserID
func (repo *chatGroupRepositoryImpl) GetMemberUserIDs(ctx context.Context, groupID int) ([]int, error) {
	var userIDs []int
	err := repo.db.WithContext(ctx).Model(&model.ChatGroupMember{}).
		Where("group_id = ?", groupID).
		Pluck("user_id", &userIDs).Error

	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询群组成员ID列表失败: %w", err))
	}
	return userIDs, nil
}

// UpdateMemberLastReadMsgID 更新成员的最后已读消息ID
func (repo *chatGroupRepositoryImpl) UpdateMemberLastReadMsgID(ctx context.Context, groupID int, userID int, messageID int64) error {
	err := repo.db.WithContext(ctx).Model(&model.ChatGroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Update("last_read_msg_id", messageID).Error

	if err != nil {
		return utils.NewSystemError(fmt.Errorf("更新成员最后已读消息ID失败: %w", err))
	}
	return nil
}

// HasUnreadMessages 检查用户是否有未读消息
func (repo *chatGroupRepositoryImpl) HasUnreadMessages(ctx context.Context, userID int) (bool, error) {
	var count int64
	err := repo.db.WithContext(ctx).
		Table("chat_group_members cgm").
		Joins("JOIN chat_groups cg ON cgm.group_id = cg.id").
		Where("cgm.user_id = ? AND cg.latest_message_id > cgm.last_read_msg_id", userID).
		Limit(1).
		Count(&count).Error

	if err != nil {
		return false, utils.NewSystemError(fmt.Errorf("查询未读消息失败: %w", err))
	}

	return count > 0, nil
}

// ListUserGroups 查询用户的所有群组列表及其相关信息
func (repo *chatGroupRepositoryImpl) ListUserGroups(ctx context.Context, userID int) ([]dto.ListUserGroupsResponse, error) {
	var results []dto.ListUserGroupsResponse

	sql := `
		SELECT
			cg.id AS group_id,
			cg.group_name,
			cg.desc,
			cm.content AS last_message,
			u.nickname AS last_message_sender,
			cm.create_time AS last_message_time,
			(cg.latest_message_id > cgm.last_read_msg_id) AS has_unread
		FROM
			chat_group_members AS cgm
		JOIN
			chat_groups AS cg ON cgm.group_id = cg.id
		LEFT JOIN
			chat_messages AS cm ON cg.latest_message_id = cm.id
		LEFT JOIN
			users AS u ON cm.sender_id = u.user_id
		WHERE
			cgm.user_id = ?
		ORDER BY
			COALESCE(cm.create_time, cg.create_time) DESC
	`

	if err := repo.db.WithContext(ctx).Raw(sql, userID).Scan(&results).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询用户群组列表失败: %w", err))
	}

	return results, nil
}

// GetMember 获取指定的群组成员信息
func (repo *chatGroupRepositoryImpl) GetMember(ctx context.Context, groupID int, userID int) (*model.ChatGroupMember, error) {
	var member model.ChatGroupMember
	if err := repo.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not a member
		}
		return nil, utils.NewSystemError(fmt.Errorf("查询群成员信息失败: %w", err))
	}
	return &member, nil
}

// ListGroupMessages 分页查询群组的消息列表
func (repo *chatGroupRepositoryImpl) ListGroupMessages(ctx context.Context, groupID int, joinMsgID int64, req dto.ListGroupMessagesReq) ([]dto.ListGroupMessagesResponse, int64, error) {
	var messages []dto.ListGroupMessagesResponse
	var total int64

	// 构建基础查询
	query := repo.db.WithContext(ctx).
		Table("chat_messages cm").
		Select("cm.id as message_id, cm.content, cm.sender_id, u.nickname as sender_name, u.avatar_url as sender_avatar, cm.send_at").
		Joins("JOIN users u ON cm.sender_id = u.user_id").
		Where("cm.group_id = ? AND cm.id > ?", groupID, joinMsgID)

	// 添加筛选条件
	if req.Content != "" {
		query = query.Where("cm.content LIKE ?", "%"+req.Content+"%")
	}
	if req.StartDate != "" {
		query = query.Where("cm.send_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		// 结束日期需要包含当天，所以查询条件是小于第二天的开始
		endDate, _ := time.Parse("2006-01-02", req.EndDate)
		query = query.Where("cm.send_at < ?", endDate.AddDate(0, 0, 1))
	}

	query = query.Order("cm.id DESC") // 先按消息ID倒序查询，拿到最新的消息在前面

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("查询群消息总数失败: %w", err))
	}

	// 应用分页和排序并查询数据
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&messages).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("分页查询群消息列表失败: %w", err))
	}

	return messages, total, nil
}

// ListMemberIDsByGroupID 根据群组ID查询所有成员的ID列表
func (repo *chatGroupRepositoryImpl) ListMemberIDsByGroupID(ctx context.Context, groupID int) ([]int, error) {
	var userIDs []int
	if err := repo.db.WithContext(ctx).Model(&model.ChatGroupMember{}).Where("group_id = ?", groupID).Pluck("user_id", &userIDs).Error; err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("查询群组成员ID列表失败: %w", err))
	}
	return userIDs, nil
}

// DeleteGroup 在事务中删除群组及其所有相关数据
func (repo *chatGroupRepositoryImpl) DeleteGroup(ctx context.Context, tx *gorm.DB, groupID int) error {
	// 1. 删除聊天记录
	if err := tx.WithContext(ctx).Where("group_id = ?", groupID).Delete(&model.ChatMessage{}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("删除群组聊天记录失败: %w", err))
	}

	// 2. 删除群成员
	if err := tx.WithContext(ctx).Where("group_id = ?", groupID).Delete(&model.ChatGroupMember{}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("删除群组成员失败: %w", err))
	}

	// 3. 删除群组本身
	if err := tx.WithContext(ctx).Delete(&model.ChatGroup{}, groupID).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("删除群组失败: %w", err))
	}

	return nil
}

// AddUserToGroup 将用户添加到群组中
func (repo *chatGroupRepositoryImpl) AddUserToGroup(ctx context.Context, groupID int, userID int) error {
	if err := repo.db.WithContext(ctx).Create(&model.ChatGroupMember{
		GroupID:  groupID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("添加用户到群组失败: %w", err))
	}
	return nil
}

// DeleteUserFromGroup 将用户从群组中移除
func (repo *chatGroupRepositoryImpl) DeleteUserFromGroup(ctx context.Context, groupID int, userID int) error {
	if err := repo.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&model.ChatGroupMember{}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("删除用户从群组失败: %w", err))
	}
	return nil
}

// ListAllGroups 列出所有群组（管理员权限）
func (repo *chatGroupRepositoryImpl) ListAllGroups(ctx context.Context, page, pageSize int) ([]model.ChatGroup, int64, error) {
	var groups []model.ChatGroup
	var total int64

	// 1. 查询总数
	if err := repo.db.WithContext(ctx).Model(&model.ChatGroup{}).Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("查询群组总数失败: %w", err))
	}

	// 2. 分页查询
	if err := repo.db.WithContext(ctx).Model(&model.ChatGroup{}).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&groups).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("分页查询群组失败: %w", err))
	}

	return groups, total, nil
}

// FindUsersNotInGroup 查询不在指定群组中的用户列表（可按姓名搜索）
func (repo *chatGroupRepositoryImpl) FindUsersNotInGroup(ctx context.Context, groupID int, page, pageSize int, name string) ([]dto.NotInGroupUserResponse, int64, error) {
	var users []dto.NotInGroupUserResponse
	var total int64

	// 构建基础查询
	query := repo.db.WithContext(ctx).Table("users u").
		Select(`u.user_id, u.nickname, u.name, u.avatar_url, u.gender as gender_code, 
			CASE 
				WHEN u.gender = 'M' THEN '男' 
				WHEN u.gender = 'F' THEN '女' 
				ELSE '未知' 
			END as gender,
			u.phone_number, u.email, u.unit, u.department, u.position, 
			u.industry, i.industry_name`).
		Joins("LEFT JOIN industries i ON u.industry = i.industry_code").
		Where(`NOT EXISTS (
			SELECT 1 
			FROM chat_group_members m 
			WHERE m.user_id = u.user_id AND m.group_id = ?
		)`, groupID)

	// 如果提供了姓名，则添加模糊查询条件
	if name != "" {
		query = query.Where("u.name LIKE ?", "%"+name+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("查询群外用户总数失败: %w", err))
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Limit(pageSize).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("查询群外用户列表失败: %w", err))
	}

	return users, total, nil
}
