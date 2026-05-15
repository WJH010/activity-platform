package repository

import (
	"context"
	"errors"
	"event-platform/internal/message/dto"
	"event-platform/internal/message/model"
	"event-platform/internal/utils"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MessageRepository 数据访问接口，定义数据访问的方法集
type MessageRepository interface {
	// GetMessageContent 内容查询
	GetMessageContent(ctx context.Context, messageID int) (*model.Message, error)
	// MarkAllMessagesAsRead 一键已读，更新所有未读消息为已读
	MarkAllMessagesAsRead(ctx context.Context, userID int) error
	// ListMessageGroupsByUserID 查询用户消息群组列表
	ListMessageGroupsByUserID(ctx context.Context, page, pageSize int, userID int, typeCode string) ([]*dto.MessageGroupDTO, int64, error)
	// ListMsgByGroups 分页查询分组内消息列表
	ListMsgByGroups(ctx context.Context, page, pageSize int, msgGroupID int, userID int) ([]*dto.ListMessageDTO, int64, error)
	// MarkAsReadByGroup 按分组更新消息为已读
	MarkAsReadByGroup(ctx context.Context, userID int, msgGroupID int)
	// CheckUserMsgPermission 权限校验查询，确保普通用户只能查看自己的消息
	CheckUserMsgPermission(ctx context.Context, userID int, msgGroupID int) error
	// HasUnreadMessages 获取是否有未读消息
	HasUnreadMessages(ctx context.Context, userID int, typeCode string) (string, error)
	// GetLatestMsgIDInGroup 获取组内最新消息ID
	GetLatestMsgIDInGroup(ctx context.Context, tx *gorm.DB, msgGroupID int) (int, error)
	// GetMaxMsgIDInGroup 获取组内最大消息ID（通过消息群组映射表查询）
	GetMaxMsgIDInGroup(ctx context.Context, tx *gorm.DB, msgGroupID int) (int, error)
	// CreateMessage 创建消息记录
	CreateMessage(ctx context.Context, tx *gorm.DB, message *model.Message) error
	// CreateMessageGroupMapping 创建消息-群组关联记录
	CreateMessageGroupMapping(ctx context.Context, tx *gorm.DB, mapping *model.MessageGroupMapping) error
	// ListMessagesByGroupID 查询指定消息组的所有消息
	ListMessagesByGroupID(ctx context.Context, page, pageSize int, msgGroupID int, title string, queryScope string) ([]*dto.ListMessageDTO, int64, error)
	// DeleteMessageGroupMapping 删除消息-群组关联记录
	DeleteMessageGroupMapping(ctx context.Context, tx *gorm.DB, mapID int, userID int) error
	// UpdateLastMsgID 更新消息群组最后一条消息ID
	UpdateLastMsgID(ctx context.Context, tx *gorm.DB, msgGroupID int, lastMsgID int) error
	// GetGroupIDByMapID 根据消息-群组关联ID查询消息群组ID
	GetGroupIDByMapID(ctx context.Context, tx *gorm.DB, mapID int) (int, int, error)
}

type GroupLatestMsg struct {
	MapID       int `gorm:"column:map_id"`
	LatestMsgID int `gorm:"column:latest_msg_id"`
}

// MessageRepositoryImpl 实现接口的具体结构体
type MessageRepositoryImpl struct {
	db *gorm.DB
}

// NewMessageRepository 创建数据访问实例
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &MessageRepositoryImpl{db: db}
}

// GetMessageContent 内容查询
func (repo *MessageRepositoryImpl) GetMessageContent(ctx context.Context, messageID int) (*model.Message, error) {
	var message model.Message

	result := repo.db.WithContext(ctx).First(&message, messageID)
	err := result.Error

	// 查询消息内容
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果消息不存在，返回业务错误
			return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "消息不存在或已被删除，请刷新页面后重试")
		}
		return nil, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return &message, nil
}

// HasUnreadMessages 获取是否有未读消息
func (repo *MessageRepositoryImpl) HasUnreadMessages(ctx context.Context, userID int, typeCode string) (string, error) {
	var count int64

	// 构建查询
	query := repo.db.WithContext(ctx).Table("user_message_groups umg").
		// 关联用户加入的非全员群组映射（全员群组无此记录）
		Joins("JOIN user_msg_group_mappings umgm ON umgm.msg_group_id = umg.id").
		Where("umg.is_deleted = ?", utils.DeletedFlagNo).                // 只查询未删除的消息分组
		Where("umgm.is_deleted = ?", utils.DeletedFlagNo).               // 确保用户在该分组内
		Where("umg.latest_msg_id > COALESCE(umgm.last_read_msg_id, 0)"). // 只查询有未读消息的分组
		Where("umgm.user_id = ?", userID)                                // 指定用户ID

	switch typeCode {
	case utils.TypeGroup:
		// 群组消息(非全员消息)
		query = query.Where("umg.include_all_user = ?", utils.FlagNo)
	case utils.TypeSystem:
		// 系统消息(全员消息)
		query = query.Where("umg.include_all_user = ?", utils.FlagYes)
	default:
		// 否则查询所有类型的未读消息
	}

	// 执行计数查询
	err := query.Count(&count).Error
	if err != nil {
		return utils.FlagNo, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	if count > 0 {
		return utils.FlagYes, nil
	} else {
		return utils.FlagNo, nil
	}
}

// MarkAllMessagesAsRead 一键已读，更新指定用户所有未读消息为已读
func (repo *MessageRepositoryImpl) MarkAllMessagesAsRead(ctx context.Context, userID int) error {
	var groupList []GroupLatestMsg
	query := repo.db.WithContext(ctx).Table("user_message_groups umg").
		Select("umgm.id AS map_id, umg.latest_msg_id").
		Joins("JOIN user_msg_group_mappings umgm ON umgm.msg_group_id = umg.id").
		Where("umg.is_deleted = ?", utils.DeletedFlagNo).
		Where("umgm.is_deleted = ?", utils.DeletedFlagNo).
		Where("umgm.user_id = ?", userID).
		Where("umgm.last_read_msg_id < umg.latest_msg_id") // 只查询有未读消息的分组

	// 执行查询，获取每个消息组的最新消息ID
	if err := query.Scan(&groupList).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("查询用户消息组失败: %w", err))
	}

	// 批量更新LastReadMsgID
	if len(groupList) == 0 {
		return nil // 无数据需更新，直接返回
	}

	// 构建 CASE 语句，批量更新所有记录
	caseStmt := "CASE id"
	args := []interface{}{}
	ids := make([]int, 0, len(groupList))
	for _, group := range groupList {
		caseStmt += " WHEN ? THEN ?"
		args = append(args, group.MapID, group.LatestMsgID)
		ids = append(ids, group.MapID)
	}
	caseStmt += " END"

	// 执行单条 SQL 批量更新
	err := repo.db.WithContext(ctx).Model(&model.UserMsgGroupMapping{}).
		Where("id IN (?)", ids). // 限制更新范围，避免误操作
		Updates(map[string]interface{}{
			"last_read_msg_id": gorm.Expr(caseStmt, args...),
			"update_user":      userID,
		}).Error

	if err != nil {
		return utils.NewSystemError(fmt.Errorf("批量更新LastReadMsgID失败: %w", err))
	}

	return nil
}

// ListMessageGroupsByUserID 查询用户消息群组列表
func (repo *MessageRepositoryImpl) ListMessageGroupsByUserID(ctx context.Context, page, pageSize int, userID int, typeCode string) ([]*dto.MessageGroupDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var results []*dto.MessageGroupDTO

	// 构建查询
	query := repo.db.WithContext(ctx).Table("user_message_groups umg").
		Select(`
            umg.id AS msg_group_id,
            umg.group_name,
            m.title AS latest_title,
            m.content AS latest_content,
            m.send_time AS latest_send_time,
            member_counts.count AS member_count,
            CASE
                WHEN umg.latest_msg_id > COALESCE(umgm.last_read_msg_id, 0) THEN 'Y'
                ELSE 'N'
            END AS has_unread
        `).
		Joins("JOIN user_msg_group_mappings umgm ON umgm.msg_group_id = umg.id").
		Joins("LEFT JOIN messages m ON m.id = umg.latest_msg_id AND m.id > umgm.join_msg_id AND m.is_deleted = ?", utils.DeletedFlagNo).
		// 成员计数子查询
		Joins(`
			JOIN (
				SELECT msg_group_id, COUNT(*) AS count 
				FROM user_msg_group_mappings 
				WHERE is_deleted = ?
				GROUP BY msg_group_id 
				) member_counts ON member_counts.msg_group_id = umg.id`, utils.DeletedFlagNo).
		Where("umg.is_deleted = ?", utils.DeletedFlagNo).  // 仅有效群组
		Where("umgm.is_deleted = ?", utils.DeletedFlagNo). // 仅有效用户-群组映射
		Where("umgm.user_id = ?", userID)

	// 根据typeCode动态拼接群组归属条件
	switch typeCode {
	case utils.TypeGroup:
		// 群组消息(非全员消息)
		query = query.Where("umg.include_all_user = ?", utils.FlagNo)
	case utils.TypeSystem:
		// 系统消息(全员消息)
		query = query.Where("umg.include_all_user = ?", utils.FlagYes)
	default:
		return nil, 0, utils.NewBusinessError(utils.ErrCodeParamInvalid, "消息类型参数不合法")
	}

	// 排序逻辑（考虑messages表无匹配数据的情况）
	query = query.Order("CASE WHEN m.send_time IS NULL THEN 0 ELSE 1 END DESC, m.send_time DESC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return results, total, nil
}

// ListMsgByGroups 分页查询分组内消息列表
func (repo *MessageRepositoryImpl) ListMsgByGroups(ctx context.Context, page, pageSize int, msgGroupID int, userID int) ([]*dto.ListMessageDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var results []*dto.ListMessageDTO
	query := repo.db.WithContext(ctx)

	// 构建基础查询
	query = query.Table("messages m").
		Select("m.id, m.title, m.content, m.send_time").
		Joins("JOIN message_group_mappings mgm ON mgm.message_id = m.id").
		Joins("JOIN user_msg_group_mappings umgm ON umgm.msg_group_id = mgm.msg_group_id AND umgm.user_id = ?", userID).
		Where("mgm.msg_group_id = ?", msgGroupID).
		Where("m.id > umgm.join_msg_id").
		Where("m.is_deleted = ?", utils.DeletedFlagNo).  // 只查询未删除的消息
		Where("mgm.is_deleted = ?", utils.DeletedFlagNo) // 只查询未删除的组内消息

	// 按发送时间降序排列
	query = query.Order("m.send_time DESC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Select("count(distinct m.id)").Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return results, total, nil
}

// MarkAsReadByGroup 更新组内消息为已读
// 根据用户ID、群组ID、更新该用户在该群组内的最后已读消息ID
func (repo *MessageRepositoryImpl) MarkAsReadByGroup(ctx context.Context, userID int, msgGroupID int) {
	// 获取组内最新消息ID
	latestMsgID, err := repo.GetLatestMsgIDInGroup(ctx, nil, msgGroupID)
	if err != nil {
		// 只记录日志，更新已读状态失败不影响消息加载
		logrus.Errorf("获取组内最新消息ID失败: %v", err)
		return
	}

	// 更新用户在该组内的最后已读消息ID
	err = repo.db.WithContext(ctx).
		Model(&model.UserMsgGroupMapping{}).
		Where("msg_group_id = ? AND user_id = ?", msgGroupID, userID).
		Updates(map[string]interface{}{
			"last_read_msg_id": latestMsgID,
			"UpdateUser":       userID,
		}).Error

	if err != nil {
		// 只记录日志，更新已读状态失败不影响消息加载
		logrus.Errorf("更新消息状态失败: %v", err)
	}
}

// GetLatestMsgIDInGroup 获取组内最新消息ID（直接查询消息群组表中最新消息字段）
func (repo *MessageRepositoryImpl) GetLatestMsgIDInGroup(ctx context.Context, tx *gorm.DB, msgGroupID int) (int, error) {
	var latestMsgID int

	if tx == nil {
		tx = repo.db.WithContext(ctx)
	}
	err := tx.Table("user_message_groups").
		Select("latest_msg_id").
		Where("id = ?", msgGroupID).
		Scan(&latestMsgID).Error

	if err != nil {
		return 0, utils.NewSystemError(fmt.Errorf("查询组内最新消息ID失败: %w", err))
	}

	return latestMsgID, nil
}

// GetMaxMsgIDInGroup 获取组内最大消息ID（通过消息群组映射表查询）
func (repo *MessageRepositoryImpl) GetMaxMsgIDInGroup(ctx context.Context, tx *gorm.DB, msgGroupID int) (int, error) {
	var maxMsgID int

	if tx == nil {
		tx = repo.db.WithContext(ctx)
	}
	err := tx.Table("message_group_mappings").
		Select("COALESCE(MAX(message_id), 0) AS max_message_id").
		Where("msg_group_id = ? AND is_deleted = ?", msgGroupID, utils.DeletedFlagNo).
		Scan(&maxMsgID).Error

	if err != nil {
		return 0, utils.NewSystemError(fmt.Errorf("查询组内最新消息ID失败: %w", err))
	}

	return maxMsgID, nil
}

// CheckUserMsgPermission 权限校验查询，确保普通用户只能查看自己的消息
func (repo *MessageRepositoryImpl) CheckUserMsgPermission(ctx context.Context, userID int, msgGroupID int) error {
	// 如果是全体用户消息组，直接返回true
	var count int64
	err := repo.db.WithContext(ctx).
		Table("user_message_groups").
		Where("id = ? AND include_all_user = ? AND is_deleted = ?", msgGroupID, utils.FlagYes, utils.DeletedFlagNo).
		Count(&count).Error
	if err != nil {
		return utils.NewSystemError(fmt.Errorf("权限校验时数据库查询失败: %w", err))
	}

	if count == 1 {
		return nil
	}

	// 如果是管理员，直接返回true
	err = repo.db.WithContext(ctx).
		Table("users").
		Where("user_id = ? AND role in (?,?)", userID, utils.RoleAdmin, utils.RoleSuperAdmin).
		Count(&count).Error
	if err != nil {
		return utils.NewSystemError(fmt.Errorf("权限校验时数据库查询失败: %w", err))
	}

	if count == 1 {
		return nil
	}

	// 检查普通用户是否属于该消息组
	err = repo.db.WithContext(ctx).
		Table("user_msg_group_mappings").
		Where("user_id = ? AND msg_group_id = ? AND is_deleted = ?", userID, msgGroupID, utils.DeletedFlagNo).
		Count(&count).Error

	if err != nil {
		return utils.NewSystemError(fmt.Errorf("权限校验时数据库查询失败: %w", err))
	}

	if count == 0 {
		return utils.NewBusinessError(utils.ErrCodePermissionDenied, "无权访问该消息组")
	}

	return nil
}

// CreateMessage 创建消息记录
func (repo *MessageRepositoryImpl) CreateMessage(ctx context.Context, tx *gorm.DB, message *model.Message) error {
	if err := tx.WithContext(ctx).Create(message).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建消息记录失败: %v", err))
	}
	return nil
}

// CreateMessageGroupMapping 创建消息-群组关联记录
func (repo *MessageRepositoryImpl) CreateMessageGroupMapping(ctx context.Context, tx *gorm.DB, mapping *model.MessageGroupMapping) error {
	if err := tx.WithContext(ctx).Create(mapping).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("创建消息-群组关联记录失败: %v", err))
	}
	return nil
}

// ListMessagesByGroupID 查询指定消息组的所有消息
func (repo *MessageRepositoryImpl) ListMessagesByGroupID(ctx context.Context, page, pageSize int, msgGroupID int, title string, queryScope string) ([]*dto.ListMessageDTO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	var results []*dto.ListMessageDTO
	query := repo.db.WithContext(ctx)

	// 构建基础查询
	query = query.Table("messages m").
		Select("m.id, mgm.id AS map_id,m.title, m.content, m.send_time").
		Joins("JOIN message_group_mappings mgm ON mgm.message_id = m.id").
		Where("mgm.msg_group_id = ?", msgGroupID).
		Where("m.is_deleted = ?", utils.DeletedFlagNo) // 只查询未删除的消息

	if queryScope != "" {
		// 如果传入了查询范围，则添加查询条件
		// 如果传入了查询范围为DELETED，则查询已删除的
		if queryScope == utils.QueryScopeDeleted {
			query = query.Where("mgm.is_deleted = ?", utils.DeletedFlagYes)
		}
		if queryScope == utils.QueryScopeAll {
			// 如果传入了查询范围为ALL，则查询所有
		}
	} else {
		// 默认查询未删除的
		query = query.Where("mgm.is_deleted = ?", utils.DeletedFlagNo)
	}

	if title != "" {
		query = query.Where("m.title LIKE ?", "%"+title+"%")
	}

	// 按发送时间降序排列
	query = query.Order("m.send_time DESC")

	// 计算总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Select("count(distinct m.id)").Count(&total).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("计算总数时数据库查询失败: %v", err))
	}

	// 查询数据
	if err := query.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, utils.NewSystemError(fmt.Errorf("数据库查询失败: %v", err))
	}

	return results, total, nil
}

// DeleteMessageGroupMapping 删除消息-群组关联记录
func (repo *MessageRepositoryImpl) DeleteMessageGroupMapping(ctx context.Context, tx *gorm.DB, mapID int, userID int) error {
	if err := tx.WithContext(ctx).Model(&model.MessageGroupMapping{}).
		Where("id = ?", mapID).
		Updates(map[string]interface{}{
			"is_deleted":  utils.DeletedFlagYes,
			"Update_user": userID,
		}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("删除消息-群组关联记录失败: %v", err))
	}
	return nil
}

// UpdateLastMsgID 更新消息群组最后一条消息ID
func (repo *MessageRepositoryImpl) UpdateLastMsgID(ctx context.Context, tx *gorm.DB, msgGroupID int, lastMsgID int) error {
	if err := tx.WithContext(ctx).Model(&model.UserMessageGroup{}).
		Where("id = ?", msgGroupID).
		Updates(map[string]interface{}{
			"latest_msg_id": lastMsgID,
		}).Error; err != nil {
		return utils.NewSystemError(fmt.Errorf("更新消息群组最后一条消息ID失败: %v", err))
	}
	return nil
}

// GetGroupIDByMapID 根据消息-群组关联ID查询消息群组ID
func (repo *MessageRepositoryImpl) GetGroupIDByMapID(ctx context.Context, tx *gorm.DB, mapID int) (int, int, error) {
	var mapping model.MessageGroupMapping
	if err := tx.WithContext(ctx).Model(&model.MessageGroupMapping{}).Where("id = ?", mapID).First(&mapping).Error; err != nil {
		return 0, 0, utils.NewSystemError(fmt.Errorf("查询消息群组ID和消息ID失败: %v", err))
	}
	return mapping.MsgGroupID, mapping.MessageID, nil
}
