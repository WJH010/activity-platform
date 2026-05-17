package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"event-platform/internal/chat/dto"
	"event-platform/internal/chat/model"
	"event-platform/internal/chat/repository"
	"event-platform/internal/database"
	user_repo "event-platform/internal/user/repository"
	"event-platform/internal/utils"
	"fmt"

	"gorm.io/gorm"
)

// 撤回消息的时间限制
const revokeTimeout = 2 * time.Minute

// ChatService 服务接口，定义方法。
type ChatService interface {
	// CreateMessage 负责解析、持久化消息，并返回可供广播的干净消息体、需要通知的用户列表和通知内容
	CreateMessage(ctx context.Context, msg *dto.WebSocketMsg) (*dto.ClientMsg, []int, []byte, error)
}

// ChatServiceImpl 实现接口的具体结构体
type ChatServiceImpl struct {
	chatRepo      repository.ChatRepository
	chatGroupRepo repository.ChatGroupRepository
	userRepo      user_repo.UserRepository
}

// NewChatService 创建服务实例
func NewChatService(chatRepo repository.ChatRepository, chatGroupRepo repository.ChatGroupRepository, userRepo user_repo.UserRepository) ChatService {
	return &ChatServiceImpl{
		chatRepo:      chatRepo,
		chatGroupRepo: chatGroupRepo,
		userRepo:      userRepo,
	}
}

// CreateMessage 负责处理并存储新的聊天消息
func (s *ChatServiceImpl) CreateMessage(ctx context.Context, msg *dto.WebSocketMsg) (*dto.ClientMsg, []int, []byte, error) {
	var clientMsg dto.ClientMsg
	// 解析客户端消息
	if err := json.Unmarshal(msg.Data, &clientMsg); err != nil {
		logrus.Errorf("解析客户端消息失败, UserID: %d, GroupID: %d, RawData: %q, Error: %v", msg.UserID, msg.GroupID, msg.Data, err)
		return nil, nil, nil, utils.NewBusinessError(utils.ErrCodeParamInvalid, "消息格式错误")
	}

	var chatMessage *model.ChatMessage

	// 开启事务
	err := database.WithTx(database.GetDB(), func(tx *gorm.DB) error {
		// 处理不同类型的聊天消息
		switch clientMsg.Type {
		// 普通聊天消息，直接存储
		case "chat":
			chatMessage = &model.ChatMessage{
				GroupID:    msg.GroupID,
				SenderID:   msg.UserID,
				SendAt:     time.Now(),
				Content:    clientMsg.Content,
				MsgType:    1,
				CreateUser: msg.UserID,
				UpdateUser: msg.UserID,
			}
		// 撤回消息，需要检查消息是否存在并是否过期
		case "revoke":
			var revokeInfo struct {
				MsgIDToRevoke int64 `json:"msg_id_to_revoke"`
			}
			// 解析撤回消息内容
			if err := json.Unmarshal([]byte(clientMsg.Content), &revokeInfo); err != nil {
				logrus.Errorf("解析撤回消息内容失败, RawContent: %s, Error: %v", clientMsg.Content, err)
				return utils.NewBusinessError(utils.ErrCodeParamInvalid, "撤回消息格式错误")
			}
			// 检查要撤回的消息是否存在
			originalMsg, err := s.chatRepo.GetMessageByID(ctx, revokeInfo.MsgIDToRevoke)
			if err != nil {
				return err
			}
			if originalMsg == nil || originalMsg.SenderID != msg.UserID {
				return utils.NewBusinessError(utils.ErrCodePermissionDenied, "消息不存在或您无权撤回")
			}
			if time.Since(originalMsg.CreateTime) > revokeTimeout {
				return utils.NewBusinessError(utils.ErrCodePermissionDenied, "超过2分钟，消息无法撤回")
			}

			if err := s.chatRepo.RevokeMessage(ctx, tx, revokeInfo.MsgIDToRevoke, msg.UserID); err != nil {
				logrus.Errorf("标记消息为已撤回失败, MsgID: %d, UserID: %d, Error: %v", revokeInfo.MsgIDToRevoke, msg.UserID, err)
				return err
			}

			chatMessage = &model.ChatMessage{
				GroupID:    msg.GroupID,
				SenderID:   msg.UserID,
				SendAt:     time.Now(),
				Content:    clientMsg.Content,
				MsgType:    3,
				CreateUser: msg.UserID,
				UpdateUser: msg.UserID,
			}
		// 其他类型的消息，直接存储
		default:
			chatMessage = &model.ChatMessage{
				GroupID:    msg.GroupID,
				SenderID:   msg.UserID,
				SendAt:     time.Now(),
				Content:    clientMsg.Content,
				MsgType:    1,
				CreateUser: msg.UserID,
				UpdateUser: msg.UserID,
			}
		}
		// 存储消息到数据库
		if err := s.chatRepo.CreateMessage(ctx, tx, chatMessage); err != nil {
			logrus.Errorf("数据库创建消息失败, UserID: %d, GroupID: %d, Error: %v", msg.UserID, msg.GroupID, err)
			return err
		}
		// 更新群组最新消息ID
		if err := s.chatGroupRepo.UpdateLatestMessageID(ctx, tx, msg.GroupID, chatMessage.ID); err != nil {
			logrus.Errorf("更新群组最新消息ID失败, GroupID: %d, MessageID: %d, Error: %v", msg.GroupID, chatMessage.ID, err)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	// 准备个人通知内容
	notificationData, memberIDs, err := s.prepareNotification(context.Background(), msg.GroupID, msg.UserID, chatMessage)
	if err != nil {
		// 个人通知的准备失败不应阻塞主流程，只记录日志
		logrus.Errorf("准备个人通知失败: %v", err)
		// 即使失败，也正常返回主消息
		clientMsg.ID = chatMessage.ID
		return &clientMsg, nil, nil, nil
	}

	// 查询发送者的昵称和头像
	sender, err := s.userRepo.GetUserByID(ctx, msg.UserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("获取发送者信息失败: %w", err)
	}
	if sender == nil {
		return nil, nil, nil, fmt.Errorf("发送者 %d 不存在", msg.UserID)
	}
	clientMsg.SenderName = sender.Nickname
	clientMsg.SenderAvatar = sender.AvatarURL

	// 持久化成功后，将消息ID填充回 clientMsg 以供广播
	clientMsg.ID = chatMessage.ID
	clientMsg.SendAt = chatMessage.SendAt
	return &clientMsg, memberIDs, notificationData, nil
}

// prepareNotification 准备新消息的个人通知内容
func (s *ChatServiceImpl) prepareNotification(ctx context.Context, groupID int, senderID int, msg *model.ChatMessage) ([]byte, []int, error) {
	// 1. 获取消息发送者信息
	sender, err := s.userRepo.GetUserByID(ctx, senderID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取发送者信息失败: %w", err)
	}
	if sender == nil {
		return nil, nil, fmt.Errorf("发送者 %d 不存在", senderID)
	}

	// 2. 获取群组所有成员ID
	memberIDs, err := s.chatGroupRepo.ListMemberIDsByGroupID(ctx, groupID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取群组成员列表失败: %w", err)
	}

	// 3. 构建通知载荷
	notification := struct {
		Event string      `json:"event"`
		Data  interface{} `json:"data"`
	}{
		Event: "new_message",
		Data: struct {
			GroupID          int       `json:"group_id"`
			LatestMessage    string    `json:"latest_message"`
			LatestSenderName string    `json:"latest_sender_name"`
			LatestSendTime   time.Time `json:"latest_send_time"`
		}{
			GroupID:          groupID,
			LatestMessage:    msg.Content, // 这里可以根据消息类型做更复杂的处理
			LatestSenderName: sender.Nickname,
			LatestSendTime:   msg.CreateTime,
		},
	}

	jsonData, err := json.Marshal(notification)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化个人通知失败: %w", err)
	}

	return jsonData, memberIDs, nil
}
