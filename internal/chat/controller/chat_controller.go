package controller

import (
	"context"
	"event-platform/internal/chat"
	"event-platform/internal/chat/dto"
	"event-platform/internal/chat/repository"
	"event-platform/internal/chat/service"
	"event-platform/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// upgrader 持有 WebSocket 的升级配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin 解决跨域问题。在生产环境中，您应该实现更严格的来源检查。
	// 例如: return r.Header.Get("Origin") == "http://your-frontend.com"
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 在生产环境中，这里应该替换为您的前端应用的实际域名
		// 例如: return r.Header.Get("Origin") == "http://www.yourapp.com"
		return true // 开发环境中暂时允许所有来源
	},
}

// ChatController 控制器
type ChatController struct {
	chatService service.ChatService
	groupRepo   repository.ChatGroupRepository
	hub         *chat.Hub
}

// NewChatController 创建控制器实例
func NewChatController(chatService service.ChatService, groupRepo repository.ChatGroupRepository, hub *chat.Hub) *ChatController {
	return &ChatController{
		chatService: chatService,
		groupRepo:   groupRepo,
		hub:         hub,
	}
}

// ServeWs 处理 websocket 请求
func (cc *ChatController) ServeWs(c *gin.Context) {

	// 1. 绑定 URL 中的 groupID
	var groupIDReq dto.GroupIDReq
	if !utils.BindUrl(c, &groupIDReq) {
		return
	}
	groupID := groupIDReq.GroupID

	// 2. 检查群组是否存在
	group, err := cc.groupRepo.GetGroupByID(c, groupID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}
	if group == nil {
		utils.HandlerFunc(c, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "群聊不存在"))
		return
	}

	// 3. 获取userID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 4. 验证用户是否为群组成员
	member, err := cc.groupRepo.GetMember(c, groupID, userID)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}
	if member == nil {
		utils.HandlerFunc(c, utils.NewBusinessError(utils.ErrCodePermissionDenied, "您不是该群组成员"))
		return
	}

	// 将 HTTP 连接升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("为用户 %d 升级 WebSocket 失败: %v", userID, err)
		return
	}

	// 创建客户端上下文
	clientCtx, cancel := context.WithCancel(context.Background())

	// 创建一个新的客户端实例
	client := &chat.Client{
		Hub:     cc.hub,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		UserID:  userID,
		GroupID: groupID,
		Ctx:     clientCtx,
		Cancel:  cancel,
	}

	// 向 Hub 注册此客户端
	client.Hub.Register <- client

	// 启动读写 goroutine
	// 这两个方法会阻塞，直到连接关闭
	go client.WritePump()
	go client.ReadPump() // 两个 pump 都在独立的 goroutine 中运行
}

// ServeNotificationWs 处理个人通知的 websocket 请求
func (cc *ChatController) ServeNotificationWs(c *gin.Context) {
	// 1. 获取 userID
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.HandlerFunc(c, err)
		return
	}

	// 2. 将 HTTP 连接升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("为用户 %d 升级个人通知 WebSocket 失败: %v", userID, err)
		return
	}

	// 3. 创建一个特殊的客户端实例（GroupID 为 0），并注册到 Hub
	clientCtx, cancel := context.WithCancel(context.Background())
	client := &chat.Client{
		Hub:     cc.hub,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		UserID:  userID,
		GroupID: 0, // GroupID 为 0 代表这是一个个人通知客户端，不属于任何特定群组
		Ctx:     clientCtx,
		Cancel:  cancel,
	}
	cc.hub.Register <- client

	// 4. 个人通知频道通常只需要被动接收消息，所以只启动 WritePump
	// ReadPump 可以在需要时启动，例如处理客户端的心跳响应
	go client.WritePump()
	// go client.ReadPump() // 暂时不启动 ReadPump，因为此频道主要用于推送
}
