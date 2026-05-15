package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"event-platform/internal/chat/dto"
	"event-platform/internal/chat/model"
	"event-platform/internal/chat/repository"
	db "event-platform/internal/database"
	"event-platform/internal/utils"
)

// ChatGroupService 定义群组业务逻辑接口
type ChatGroupService interface {
	CreateGroup(ctx context.Context, req *model.ChatGroup, creatorID int) (*model.ChatGroup, error)
	AddMembers(ctx context.Context, groupID int, req *dto.AddMembersReq, operatorID int) error
	RemoveMembers(ctx context.Context, groupID int, userIDs []int, operatorID int) error
	DeleteGroup(ctx context.Context, groupID int, operatorID int) error
	ListGroupMembers(ctx context.Context, groupID int, page, pageSize int) ([]dto.ListGroupMembersResponse, int64, error)
	HasUnreadMessages(ctx context.Context, userID int) (bool, error)
	ListUserGroups(ctx context.Context, userID int) ([]dto.ListUserGroupsResponse, error)
	ListGroupMessages(ctx context.Context, groupID int, userID int, req dto.ListGroupMessagesReq) ([]dto.ListGroupMessagesResponse, int64, error)
	GetGroupByID(ctx context.Context, groupID int) (*model.ChatGroup, error)
	AddUserToGroup(ctx context.Context, groupID int, userID int) error
	DeleteUserFromGroup(ctx context.Context, groupID int, userID int) error
	ListAllGroups(ctx context.Context, page, pageSize int) ([]model.ChatGroup, int64, error)
	FindUsersNotInGroup(ctx context.Context, groupID int, page, pageSize int, name string) ([]dto.NotInGroupUserResponse, int64, error)
}

// chatGroupServiceImpl 实现接口
type chatGroupServiceImpl struct {
	chatGroupRepo repository.ChatGroupRepository
}

// NewChatGroupService 创建实例
func NewChatGroupService(chatGroupRepo repository.ChatGroupRepository) ChatGroupService {
	return &chatGroupServiceImpl{chatGroupRepo: chatGroupRepo}
}

// CreateGroup 创建群组并添加创建者为群主
func (svc *chatGroupServiceImpl) CreateGroup(ctx context.Context, req *model.ChatGroup, creatorID int) (*model.ChatGroup, error) {
	// 准备数据模型
	group := &model.ChatGroup{
		GroupName:  req.GroupName,
		Desc:       req.Desc,
		OwnerID:    creatorID,
		CreateUser: creatorID,
		UpdateUser: creatorID,
	}

	// 开启事务
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return nil, utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	// 使用 defer-recover 模式确保事务在 panic 时也能回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	// 1. 创建群组
	if err := svc.chatGroupRepo.CreateGroup(ctx, tx, group); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 2. 创建群主成员
	member := &model.ChatGroupMember{
		GroupID:    int(group.ID),
		UserID:     creatorID,
		CreateUser: creatorID,
		UpdateUser: creatorID,
	}
	if err := svc.chatGroupRepo.CreateMember(ctx, tx, member); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}

	return group, nil
}

// AddMembers 向群组批量添加新成员
func (svc *chatGroupServiceImpl) AddMembers(ctx context.Context, groupID int, req *dto.AddMembersReq, operatorID int) error {
	// 1. 检查群组是否存在，并验证操作者是否为群主
	group, err := svc.chatGroupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "群组不存在")
	}
	if group.OwnerID != operatorID {
		return utils.NewBusinessError(utils.ErrCodePermissionDenied, "只有群主才能添加成员")
	}

	// 2. 根据 WithHistory 参数确定 JoinMsgID
	var joinMsgID int64
	if req.WithHistory != "Y" {
		// 不附带历史消息，JoinMsgID 设置为群组当前的最新消息ID
		joinMsgID = group.LatestMessageID
	}
	// 如果 req.WithHistory == "Y"，joinMsgID 默认为 0，表示可以查看所有历史消息

	// 3. 获取当前所有成员，用于去重
	existingMemberIDs, err := svc.chatGroupRepo.GetMemberUserIDs(ctx, groupID)
	if err != nil {
		return err
	}
	existingMemberSet := make(map[int]struct{}, len(existingMemberIDs))
	for _, id := range existingMemberIDs {
		existingMemberSet[id] = struct{}{}
	}

	// 4. 过滤掉已经是成员的用户，并准备新成员列表
	var newMembers []*model.ChatGroupMember
	for _, userID := range req.UserIDs {
		if _, exists := existingMemberSet[userID]; !exists {
			newMembers = append(newMembers, &model.ChatGroupMember{
				GroupID:    groupID,
				UserID:     userID,
				JoinedAt:   time.Now(),
				JoinMsgID:  joinMsgID,
				CreateUser: operatorID,
				UpdateUser: operatorID,
			})
		}
	}

	// 5. 如果没有需要添加的新成员，直接返回成功
	if len(newMembers) == 0 {
		return nil
	}

	// 6. 批量插入新成员（在事务中）
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	if err := svc.chatGroupRepo.AddMembers(ctx, tx, newMembers); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}

	return nil
}

// RemoveMembers 从群组中移除成员
func (svc *chatGroupServiceImpl) RemoveMembers(ctx context.Context, groupID int, userIDs []int, operatorID int) error {
	// 1. 检查群组是否存在，并验证操作者是否为群主
	group, err := svc.chatGroupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "群组不存在")
	}
	if group.OwnerID != operatorID {
		return utils.NewBusinessError(utils.ErrCodePermissionDenied, "只有群主才能移除成员")
	}

	// 2. 过滤掉群主，防止群主将自己移除
	var membersToRemove []int
	for _, userID := range userIDs {
		if userID != operatorID {
			membersToRemove = append(membersToRemove, userID)
		}
	}

	if len(membersToRemove) == 0 {
		return nil
	}

	// 3. 在事务中执行移除操作
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	if err := svc.chatGroupRepo.RemoveMembers(ctx, tx, groupID, membersToRemove); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}

	return nil
}

// ListGroupMembers 分页查询群组成员
func (svc *chatGroupServiceImpl) ListGroupMembers(ctx context.Context, groupID int, page, pageSize int) ([]dto.ListGroupMembersResponse, int64, error) {
	return svc.chatGroupRepo.ListGroupMembers(ctx, groupID, page, pageSize)
}

// HasUnreadMessages 检查用户是否有未读消息
func (svc *chatGroupServiceImpl) HasUnreadMessages(ctx context.Context, userID int) (bool, error) {
	return svc.chatGroupRepo.HasUnreadMessages(ctx, userID)
}

// ListUserGroups 查询用户的所有群组列表
func (svc *chatGroupServiceImpl) ListUserGroups(ctx context.Context, userID int) ([]dto.ListUserGroupsResponse, error) {
	return svc.chatGroupRepo.ListUserGroups(ctx, userID)
}

// ListGroupMessages 查询指定群组的消息列表，并在查询后将该群组标记为已读
func (svc *chatGroupServiceImpl) ListGroupMessages(ctx context.Context, groupID int, userID int, req dto.ListGroupMessagesReq) ([]dto.ListGroupMessagesResponse, int64, error) {
	// 1. 验证用户是否为群组成员，并获取其 JoinMsgID
	member, err := svc.chatGroupRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		return nil, 0, err // GetMember 内部已封装错误
	}
	if member == nil {
		return nil, 0, utils.NewBusinessError(utils.ErrCodePermissionDenied, "您不是该群组成员，无权查看消息")
	}

	// 2. 调用 repository 查询消息列表
	messages, total, err := svc.chatGroupRepo.ListGroupMessages(ctx, groupID, member.JoinMsgID, req)
	if err != nil {
		return nil, 0, err
	}

	// 3. 异步更新用户的已读消息ID到当前群组的最新消息ID
	// 我们只在用户请求第一页消息时更新，这通常意味着用户刚刚进入聊天室
	if req.Page == 1 {
		go func() {
			// 创建一个新的后台上下文，避免受到原始请求取消的影响
			bgCtx := context.Background()
			group, err := svc.chatGroupRepo.GetGroupByID(bgCtx, groupID)
			if err != nil {
				logrus.Errorf("异步更新已读状态时获取群组信息失败: %v", err)
				return
			}
			if group != nil && group.LatestMessageID > 0 && group.LatestMessageID > member.LastReadMsgID {
				if err := svc.chatGroupRepo.UpdateMemberLastReadMsgID(bgCtx, groupID, userID, group.LatestMessageID); err != nil {
					logrus.Errorf("异步更新用户 %d 在群组 %d 的已读状态失败: %v", userID, groupID, err)
				}
			}
		}()
	}

	return messages, total, nil
}

// DeleteGroup 删除群组
func (svc *chatGroupServiceImpl) DeleteGroup(ctx context.Context, groupID int, operatorID int) error {
	// 1. 检查群组是否存在，并验证操作者是否为群主
	group, err := svc.chatGroupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "群组不存在")
	}
	if group.OwnerID != operatorID {
		return utils.NewBusinessError(utils.ErrCodePermissionDenied, "只有群主才能解散群组")
	}

	// 2. 在事务中执行删除操作
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return utils.NewSystemError(fmt.Errorf("开启事务失败: %w", tx.Error))
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logrus.Panic("事务回滚，发生异常: ", r)
		}
	}()

	if err := svc.chatGroupRepo.DeleteGroup(ctx, tx, groupID); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return utils.NewSystemError(fmt.Errorf("提交事务失败: %w", err))
	}

	return nil
}

// GetGroupByID 根据ID查询群组信息
func (svc *chatGroupServiceImpl) GetGroupByID(ctx context.Context, groupID int) (*model.ChatGroup, error) {
	return svc.chatGroupRepo.GetGroupByID(ctx, groupID)
}

// AddUserToGroup 将用户添加到群组中
func (svc *chatGroupServiceImpl) AddUserToGroup(ctx context.Context, groupID int, userID int) error {
	return svc.chatGroupRepo.AddUserToGroup(ctx, groupID, userID)
}

// DeleteUserFromGroup 将用户从群组中移除
func (svc *chatGroupServiceImpl) DeleteUserFromGroup(ctx context.Context, groupID int, userID int) error {
	return svc.chatGroupRepo.DeleteUserFromGroup(ctx, groupID, userID)
}

// ListAllGroups 分页查询所有群组（管理员接口）
func (svc *chatGroupServiceImpl) ListAllGroups(ctx context.Context, page, pageSize int) ([]model.ChatGroup, int64, error) {
	return svc.chatGroupRepo.ListAllGroups(ctx, page, pageSize)
}

// FindUsersNotInGroup 查询不在指定群组中的用户列表
func (svc *chatGroupServiceImpl) FindUsersNotInGroup(ctx context.Context, groupID int, page, pageSize int, name string) ([]dto.NotInGroupUserResponse, int64, error) {
	return svc.chatGroupRepo.FindUsersNotInGroup(ctx, groupID, page, pageSize, name)
}
