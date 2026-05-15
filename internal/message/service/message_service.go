package service

import (
	"context"
	"event-platform/internal/message/dto"
	"event-platform/internal/message/model"
	"event-platform/internal/message/repository"

	"gorm.io/gorm"
)

// MessageService 服务接口，定义方法，接收 context.Context 和数据模型。
type MessageService interface {
	// GetMessageContent 获取消息内容
	GetMessageContent(ctx context.Context, messageID int) (*model.Message, error)
	// MarkAllMessagesAsRead 一键已读，更新所有未读消息为已读
	MarkAllMessagesAsRead(ctx context.Context, userID int) error
	// ListMessageGroupsByUserID 分页查询用户消息群组列表
	ListMessageGroupsByUserID(ctx context.Context, page, pageSize int, userID int, typeCode string) ([]*dto.MessageGroupDTO, int64, error)
	// ListMsgByGroups 分页查询分组内消息列表
	ListMsgByGroups(ctx context.Context, page, pageSize int, groupID int, userID int) ([]*dto.ListMessageDTO, int64, error)
	// HasUnreadMessages 检查用户是否有未读消息
	HasUnreadMessages(ctx context.Context, userID int, typeCode string) (string, error)
	// SendMessage 发送消息
	SendMessage(ctx context.Context, msgGroupID int, msg *model.Message) error
	// ListMessagesByGroupID 根据消息群组ID查询消息列表
	ListMessagesByGroupID(ctx context.Context, page, pageSize int, groupID int, title string, queryScope string) ([]*dto.ListMessageDTO, int64, error)
	// RevokeGroupMessage 撤回群组消息
	RevokeGroupMessage(ctx context.Context, mapID int, userID int) error
}

// MessageServiceImpl 实现接口的具体结构体，持有数据访问层接口 Repository 的实例
type MessageServiceImpl struct {
	messageRepo repository.MessageRepository
	groupRepo   repository.MsgGroupRepository
}

// NewMessageService 创建服务实例
func NewMessageService(messageRepo repository.MessageRepository, groupRepo repository.MsgGroupRepository) MessageService {
	return &MessageServiceImpl{messageRepo: messageRepo, groupRepo: groupRepo}
}

// GetMessageContent 获取消息内容
func (svc *MessageServiceImpl) GetMessageContent(ctx context.Context, messageID int) (*model.Message, error) {
	return svc.messageRepo.GetMessageContent(ctx, messageID)
}

// HasUnreadMessages 检查用户是否有未读消息
func (svc *MessageServiceImpl) HasUnreadMessages(ctx context.Context, userID int, typeCode string) (string, error) {
	return svc.messageRepo.HasUnreadMessages(ctx, userID, typeCode)
}

// MarkAllMessagesAsRead 一键已读，更新所有未读消息为已读
func (svc *MessageServiceImpl) MarkAllMessagesAsRead(ctx context.Context, userID int) error {
	return svc.messageRepo.MarkAllMessagesAsRead(ctx, userID)
}

// ListMessageGroupsByUserID 分页查询用户消息群组列表
func (svc *MessageServiceImpl) ListMessageGroupsByUserID(ctx context.Context, page, pageSize int, userID int, typeCode string) ([]*dto.MessageGroupDTO, int64, error) {
	return svc.messageRepo.ListMessageGroupsByUserID(ctx, page, pageSize, userID, typeCode)
}

// ListMsgByGroups 分页查询分组内消息列表
func (svc *MessageServiceImpl) ListMsgByGroups(ctx context.Context, page, pageSize int, groupID int, userID int) ([]*dto.ListMessageDTO, int64, error) {
	// 校验权限，确保普通用户只能查看自己的消息
	err := svc.messageRepo.CheckUserMsgPermission(ctx, userID, groupID)
	if err != nil {
		return nil, 0, err
	}
	// 标记组内所有消息为已读
	svc.messageRepo.MarkAsReadByGroup(ctx, userID, groupID)

	return svc.messageRepo.ListMsgByGroups(ctx, page, pageSize, groupID, userID)
}

// SendMessage 发送消息
func (svc *MessageServiceImpl) SendMessage(ctx context.Context, msgGroupID int, msg *model.Message) error {
	// 使用 GORM 函数式事务执行
	err := svc.groupRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 创建消息
		err := svc.messageRepo.CreateMessage(ctx, tx, msg)
		if err != nil {
			return err
		}

		// 创建消息-群组关联
		mapping := &model.MessageGroupMapping{
			MessageID:  msg.ID,
			MsgGroupID: msgGroupID,
			CreateUser: msg.CreateUser,
			UpdateUser: msg.CreateUser,
		}
		err = svc.messageRepo.CreateMessageGroupMapping(ctx, tx, mapping)
		if err != nil {
			return err
		}

		// 更新群组最后一条消息ID
		err = svc.messageRepo.UpdateLastMsgID(ctx, tx, msgGroupID, msg.ID)
		if err != nil {
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

// ListMessagesByGroupID 根据消息群组ID查询消息列表
func (svc *MessageServiceImpl) ListMessagesByGroupID(ctx context.Context, page, pageSize int, groupID int, title string, queryScope string) ([]*dto.ListMessageDTO, int64, error) {
	return svc.messageRepo.ListMessagesByGroupID(ctx, page, pageSize, groupID, title, queryScope)
}

// RevokeGroupMessage 撤回群组消息
func (svc *MessageServiceImpl) RevokeGroupMessage(ctx context.Context, mapID int, userID int) error {
	// 使用 GORM 函数式事务执行
	err := svc.groupRepo.ExecTransaction(ctx, func(tx *gorm.DB) error {
		// 查询消息群组ID和消息ID
		msgGroupID, msgID, err := svc.messageRepo.GetGroupIDByMapID(ctx, tx, mapID)
		if err != nil {
			return err
		}

		// 删除消息-群组关联记录
		err = svc.messageRepo.DeleteMessageGroupMapping(ctx, tx, mapID, userID)
		if err != nil {
			return err
		}

		// 查询群组最后一条消息ID
		latestMsgID, err := svc.messageRepo.GetLatestMsgIDInGroup(ctx, tx, msgGroupID)
		if err != nil {
			return err
		}

		// 如果撤回的消息是最新消息，更新群组最后一条消息ID
		if msgID == latestMsgID {
			// 查询组内有效的最大消息ID
			maxMsgID, err := svc.messageRepo.GetMaxMsgIDInGroup(ctx, tx, msgGroupID)
			if err != nil {
				return err
			}
			err = svc.messageRepo.UpdateLastMsgID(ctx, tx, msgGroupID, maxMsgID)
			if err != nil {
				return err
			}
		}

		return nil
	})

	// 处理事务执行结果
	if err != nil {
		return err
	}
	return nil
}
