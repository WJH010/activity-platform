package service

import (
	"activity-platform/internal/message/dto"
	"activity-platform/internal/message/model"
	"activity-platform/internal/message/repository"
	"activity-platform/internal/utils"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type MsgGroupService interface {
	// GetMsgGroupByID 根据id获取消息群组信息
	GetMsgGroupByID(ctx context.Context, msgGroupID int) (*model.UserMessageGroup, error)
	// AddUserToGroup 用户入群
	AddUserToGroup(ctx context.Context, msgGroupID int, userIDs []int, operateUser int) error
	// CreateMsgGroup 创建消息群组
	CreateMsgGroup(ctx context.Context, msgGroup *model.UserMessageGroup, userIDs []int) error
	// DeleteUserFromGroup 用户退群
	DeleteUserFromGroup(ctx context.Context, msgGroupID int, userIDs []int, operateUser int) error
	// UpdateMsgGroup 更新消息群组
	UpdateMsgGroup(ctx context.Context, msgGroupID int, request dto.UpdateMsgGroupRequest, userID int) error
	// DeleteMsgGroup 删除消息群组
	DeleteMsgGroup(ctx context.Context, msgGroupID int, userID int) error
	// ListMsgGroups 列表查询消息群组
	ListMsgGroups(ctx context.Context, page int, pageSize int, req dto.ListMsgGroupRequest) ([]dto.ListMsgGroupResponse, int64, error)
	// GetMsgGroupByEventID 根据活动ID查询消息群组
	GetMsgGroupByEventID(ctx context.Context, eventID int) (*model.UserMessageGroup, error)
	// ListGroupsUsers 获取指定群组内用户
	ListGroupsUsers(ctx context.Context, page int, pageSize int, msgGroupID int) ([]dto.ListGroupsUsersResponse, int64, error)
	// ListNotInGroupUsers 查询不在指定组内的用户
	ListNotInGroupUsers(ctx context.Context, page int, pageSize int, msgGroupID int, req dto.ListNotInGroupUsersRequest) ([]dto.ListGroupsUsersResponse, int64, error)
	// AddUserToAllUserGroups 将指定用户加入到包含全体用户的群组中，用于新用户注册时加入全体用户群组
	AddUserToAllUserGroups(ctx context.Context, userID int)
}

// MsgGroupServiceImpl 实现接口的具体结构体，持有数据访问层接口 Repository 的实例
type MsgGroupServiceImpl struct {
	msgGroupRepo repository.MsgGroupRepository
	msgRepo      repository.MessageRepository
}

// NewMsgGroupService 创建服务实例
func NewMsgGroupService(msgGroupRepo repository.MsgGroupRepository, msgRepo repository.MessageRepository) MsgGroupService {
	return &MsgGroupServiceImpl{msgGroupRepo: msgGroupRepo, msgRepo: msgRepo}
}

// 工具函数：分类用户（需要新增/需要恢复/无需操作）
func classifyUsers(
	userIDs []int,
	existingMappings map[int]model.UserMsgGroupMapping, // key: userID, value: 已有记录
	groupID, latestMsgID, operateUser int,
) (needCreate []model.UserMsgGroupMapping, needRecover []model.UserMsgGroupMapping) {
	for _, userID := range userIDs {
		if mapping, exists := existingMappings[userID]; exists {
			// 记录存在：若is_deleted=Y则需要恢复，否则无需操作
			if mapping.IsDeleted == utils.DeletedFlagYes {
				needRecover = append(needRecover, mapping)
			}
		} else {
			// 记录不存在：需要新增
			needCreate = append(needCreate, model.UserMsgGroupMapping{
				MsgGroupID:    groupID,
				UserID:        userID,
				JoinMsgID:     latestMsgID, // 入群时的最新消息ID（用于过滤历史消息）
				LastReadMsgID: latestMsgID,
				CreateUser:    operateUser,
				UpdateUser:    operateUser,
				IsDeleted:     utils.DeletedFlagNo,
			})
		}
	}
	return
}

// 工具函数：从需要恢复的记录中提取用户ID
func extractUserIDs(mappings []model.UserMsgGroupMapping) []int {
	var ids []int
	for _, m := range mappings {
		ids = append(ids, m.UserID)
	}
	return ids
}

// GetMsgGroupByID 根据id获取消息群组信息
func (svc *MsgGroupServiceImpl) GetMsgGroupByID(ctx context.Context, msgGroupID int) (*model.UserMessageGroup, error) {
	return svc.msgGroupRepo.GetMsgGroupByID(ctx, msgGroupID)
}

// AddUserToGroup 用户入群
func (svc *MsgGroupServiceImpl) AddUserToGroup(ctx context.Context, msgGroupID int, userIDs []int, operateUser int) error {
	// 检查群组是否存在
	group, err := svc.msgGroupRepo.GetMsgGroupByID(ctx, msgGroupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "数据异常，消息群组不存在")
	}
	// 获取当前群组最新消息ID
	latestMsgID, err := svc.msgRepo.GetLatestMsgIDInGroup(ctx, nil, msgGroupID)
	if err != nil {
		return err
	}

	// 查询已存在的用户-群组关联记录（过滤无效用户ID）
	existingMappings, err := svc.msgGroupRepo.GetExistingMappings(ctx, msgGroupID, userIDs)
	if err != nil {
		return fmt.Errorf("查询已有记录失败: %w", err)
	}

	// 分类处理：区分需要新增、需要恢复（is_deleted=Y）、无需操作（is_deleted=N）的用户
	needCreate, needRecover := classifyUsers(userIDs, existingMappings, msgGroupID, latestMsgID, operateUser)

	// 使用 GORM 函数式事务执行
	err = svc.msgGroupRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 事务内的业务操作：所有数据库操作必须使用 tx 作为 DB 实例
		if len(needCreate) > 0 {
			// 批量新增用户-群组关联记录
			if err := svc.msgGroupRepo.CreateUserMsgGroupMappings(ctx, tx, needCreate); err != nil {
				return err // 返回错误，GORM 自动回滚
			}
		}

		if len(needRecover) > 0 {
			// 批量恢复用户-群组关联记录
			userIDsToRecover := extractUserIDs(needRecover)
			if err := svc.msgGroupRepo.RecoverUserMsgGroupMappings(ctx, tx, msgGroupID, userIDsToRecover, latestMsgID, operateUser); err != nil {
				return err // 返回错误，GORM 自动回滚
			}
		}

		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}

	return nil
}

// AddUserToAllUserGroups 将指定用户加入到包含全体用户的群组中，用于新用户注册时加入全体用户群组
func (svc *MsgGroupServiceImpl) AddUserToAllUserGroups(ctx context.Context, userID int) {
	groupIDs, err := svc.msgGroupRepo.GetAllUserGroupIDs(ctx)
	if err != nil {
		// 只记录错误日志，避免影响新用户注册过程
		logrus.Errorf("获取包含全体用户的群组列表失败： %s", err.Error())
	}
	var mapings []model.UserMsgGroupMapping
	for _, groupID := range groupIDs {
		// 获取当前群组最新消息ID
		latestMsgID, err := svc.msgRepo.GetLatestMsgIDInGroup(ctx, nil, groupID)
		if err != nil {
			logrus.Errorf("获取群组最新消息ID失败： %s", err.Error())
		}
		// 拼接新建数据
		mapings = append(mapings, model.UserMsgGroupMapping{
			MsgGroupID:    groupID,
			UserID:        userID,
			JoinMsgID:     latestMsgID, // 入群时的最新消息ID（用于过滤历史消息）
			LastReadMsgID: latestMsgID,
			CreateUser:    0, // 自动创建
			UpdateUser:    0,
			IsDeleted:     utils.DeletedFlagNo,
		})
	}
	err = svc.msgGroupRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		if err := svc.msgGroupRepo.CreateUserMsgGroupMappings(ctx, tx, mapings); err != nil {
			logrus.Errorf("用户入群失败： %s", err.Error())
		}
		return nil
	})
	// 处理事务执行结果
	if err != nil {
		logrus.Errorf("事务执行失败： %s", err.Error())
	}
}

// CreateMsgGroup 创建消息群组
// 没有进行事务控制，允许群组创建成功但用户添加失败
func (svc *MsgGroupServiceImpl) CreateMsgGroup(ctx context.Context, msgGroup *model.UserMessageGroup, userIDs []int) error {
	// 创建消息群组
	err := svc.msgGroupRepo.CreateMsgGroup(ctx, msgGroup)
	if err != nil {
		return err
	}
	// 如果群组创建成功且包含用户，则添加用户到群组
	if msgGroup.IncludeAllUser == utils.FlagNo && len(userIDs) > 0 {
		err = svc.AddUserToGroup(ctx, msgGroup.ID, userIDs, msgGroup.CreateUser)
		if err != nil {
			logrus.Errorf("添加用户到群组失败 %s", err.Error())
			return utils.NewBusinessError(utils.ErrCodeServerInternalError, "添加用户到群组失败，请手动添加")
		}
	}
	// 如果是包含全体用户，则将全体用户添加到群组
	if msgGroup.IncludeAllUser == utils.FlagYes {
		var UIDs []int
		page := 1
		// 避免数据量过大，采用循环分批处理方式
		for {
			UIDs, err = svc.msgGroupRepo.GetAllUserIDs(ctx, page)
			if err != nil {
				logrus.Errorf("添加用户到群组失败 %s", err.Error())
				return utils.NewBusinessError(utils.ErrCodeServerInternalError, "添加用户到群组失败，请手动添加")
			}
			// 若没有更多数据，退出循环
			if len(UIDs) == 0 {
				break
			}
			// 批量入群
			err = svc.AddUserToGroup(ctx, msgGroup.ID, UIDs, msgGroup.CreateUser)
			if err != nil {
				logrus.Errorf("添加用户到群组失败 %s", err.Error())
				return utils.NewBusinessError(utils.ErrCodeServerInternalError, "添加用户到群组失败，请手动添加")
			}
			page++
		}

	}

	return nil
}

// DeleteUserFromGroup 用户退群
func (svc *MsgGroupServiceImpl) DeleteUserFromGroup(ctx context.Context, msgGroupID int, userIDs []int, operateUser int) error {
	// 检查群组是否存在
	group, err := svc.msgGroupRepo.GetMsgGroupByID(ctx, msgGroupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "数据异常，消息群组不存在")
	}
	// 删除用户-群组关联记录（软删除）
	err = svc.msgGroupRepo.DeleteUserMsgGroupMappings(ctx, msgGroupID, userIDs, operateUser)
	if err != nil {
		return err
	}
	return nil
}

// UpdateMsgGroup 更新消息群组
func (svc *MsgGroupServiceImpl) UpdateMsgGroup(ctx context.Context, msgGroupID int, request dto.UpdateMsgGroupRequest, userID int) error {
	// 检查群组是否存在
	group, err := svc.msgGroupRepo.GetMsgGroupByID(ctx, msgGroupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "数据异常，消息群组不存在")
	}
	// 构建更新字段
	updateField := make(map[string]interface{})
	if request.GroupName != nil {
		updateField["group_name"] = request.GroupName
	}
	if request.Desc != nil {
		updateField["desc"] = request.Desc
	}

	updateField["update_user"] = userID

	// 更新消息群组信息
	err = svc.msgGroupRepo.UpdateMsgGroup(ctx, nil, msgGroupID, updateField)
	if err != nil {
		return err
	}
	return nil
}

// DeleteMsgGroup 删除消息群组
func (svc *MsgGroupServiceImpl) DeleteMsgGroup(ctx context.Context, msgGroupID int, userID int) error {
	// 检查群组是否存在
	group, err := svc.msgGroupRepo.GetMsgGroupByID(ctx, msgGroupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "数据异常，消息群组不存在")
	}

	// 使用 GORM 函数式事务执行
	err = svc.msgGroupRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 事务内的业务操作：所有数据库操作必须使用 tx 作为 DB 实例
		// 软删除消息群组，复用 UpdateMsgGroup 方法
		updateField := map[string]interface{}{
			"is_deleted":  utils.DeletedFlagYes,
			"update_user": userID,
		}
		if err = svc.msgGroupRepo.UpdateMsgGroup(ctx, tx, msgGroupID, updateField); err != nil {
			return err
		}

		// 同步删除群组内的全部用户
		if err = svc.msgGroupRepo.DeleteUserByGroupID(ctx, tx, msgGroupID, updateField); err != nil {
			return err
		}

		return nil // 返回 nil，GORM 自动提交
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}

	return nil
}

// ListMsgGroups 列表查询消息群组
func (svc *MsgGroupServiceImpl) ListMsgGroups(ctx context.Context, page int, pageSize int, req dto.ListMsgGroupRequest) ([]dto.ListMsgGroupResponse, int64, error) {
	return svc.msgGroupRepo.ListMsgGroups(ctx, page, pageSize, req)
}

// GetMsgGroupByEventID 根据活动ID查询消息群组
func (svc *MsgGroupServiceImpl) GetMsgGroupByEventID(ctx context.Context, eventID int) (*model.UserMessageGroup, error) {
	return svc.msgGroupRepo.GetMsgGroupByEventID(ctx, eventID)
}

// ListGroupsUsers 获取指定群组内用户
func (svc *MsgGroupServiceImpl) ListGroupsUsers(ctx context.Context, page int, pageSize int, msgGroupID int) ([]dto.ListGroupsUsersResponse, int64, error) {
	return svc.msgGroupRepo.ListGroupsUsers(ctx, page, pageSize, msgGroupID)
}

// ListNotInGroupUsers 查询不在指定组内的用户
func (svc *MsgGroupServiceImpl) ListNotInGroupUsers(ctx context.Context, page int, pageSize int, msgGroupID int, req dto.ListNotInGroupUsersRequest) ([]dto.ListGroupsUsersResponse, int64, error) {
	return svc.msgGroupRepo.ListNotInGroupUsers(ctx, page, pageSize, msgGroupID, req)
}
