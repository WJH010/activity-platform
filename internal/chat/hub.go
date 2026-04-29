package chat

import (
	"activity-platform/internal/chat/dto"
	"activity-platform/internal/chat/service"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket" // <-- 已更换
	"github.com/sirupsen/logrus"
)

const (
	// writeWait 是允许向对端写入消息的时间。
	writeWait = 10 * time.Second

	// pongWait 是允许从对端读取下一个 pong 消息的时间。
	pongWait = 60 * time.Second

	// pingPeriod 是向对端发送 ping 的周期。必须小于 pongWait。
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize 是允许从对端接收的最大消息大小。
	maxMessageSize = 2048 // <-- 已加回
)

// Client 单个WebSocket连接实例
type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn // <-- 类型来自 gorilla/websocket
	Send    chan []byte
	UserID  int
	GroupID int
	Ctx     context.Context
	Cancel  context.CancelFunc
}

// Hub 的定义保持不变...
// BroadcastPayload 是在 Hub 内部广播频道中传递的数据结构。
// 它包含了所有广播所需的信息。
type BroadcastPayload struct {
	UserID  int
	GroupID int
	Data    []byte // 序列化后的 JSON 数据
}

// DirectMessagePayload 是用于个人通知的载荷
type DirectMessagePayload struct {
	TargetUserID int
	Data         []byte // 序列化后的 JSON 数据
}

// Hub 的定义
type Hub struct {
	Groups        map[int]map[*Client]bool
	UserClients   map[int]*Client // 新增：按 UserID 存储的个人通知客户端
	Register      chan *Client
	Unregister    chan *Client
	Broadcast     chan *BroadcastPayload
	DirectMessage chan *DirectMessagePayload // 新增：用于发送个人通知的通道
	Mu            sync.RWMutex
	ChatService   service.ChatService
}

// NewHub 创建 Hub 实例
func NewHub(chatService service.ChatService) *Hub {
	return &Hub{
		Groups:        make(map[int]map[*Client]bool),
		UserClients:   make(map[int]*Client),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		Broadcast:     make(chan *BroadcastPayload, 1024),
		DirectMessage: make(chan *DirectMessagePayload, 1024),
		ChatService:   chatService,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mu.Lock()
			if client.GroupID == 0 {
				// 这是一个个人通知客户端
				h.UserClients[client.UserID] = client
				logrus.Infof("个人通知客户端 %d 已注册", client.UserID)
			} else {
				// 这是一个群聊客户端
				if _, ok := h.Groups[client.GroupID]; !ok {
					h.Groups[client.GroupID] = make(map[*Client]bool)
				}
				h.Groups[client.GroupID][client] = true
				logrus.Infof("群聊客户端 %d 注册到群组 %d", client.UserID, client.GroupID)
			}
			h.Mu.Unlock()

		case client := <-h.Unregister:
			h.Mu.Lock()
			if client.GroupID == 0 {
				// 个人通知客户端注销
				if _, ok := h.UserClients[client.UserID]; ok {
					delete(h.UserClients, client.UserID)
					close(client.Send)
					logrus.Infof("个人通知客户端 %d 已注销", client.UserID)
				}
			} else {
				// 群聊客户端注销
				if _, ok := h.Groups[client.GroupID]; ok {
					if _, ok := h.Groups[client.GroupID][client]; ok {
						delete(h.Groups[client.GroupID], client)
						close(client.Send)
						if len(h.Groups[client.GroupID]) == 0 {
							delete(h.Groups, client.GroupID)
						}
						logrus.Infof("群聊客户端 %d 从群组 %d 注销", client.UserID, client.GroupID)
					}
				}
			}
			h.Mu.Unlock()

		case payload := <-h.Broadcast:
			h.Mu.RLock()
			if clients, ok := h.Groups[payload.GroupID]; ok {
				for client := range clients {
					select {
					case client.Send <- payload.Data:
					default:
						close(client.Send)
						delete(clients, client)
					}
				}
			}
			h.Mu.RUnlock()

		case payload := <-h.DirectMessage:
			h.Mu.RLock()
			if client, ok := h.UserClients[payload.TargetUserID]; ok {
				select {
				case client.Send <- payload.Data:
				default:
					close(client.Send)
					delete(h.UserClients, payload.TargetUserID)
				}
			}
			h.Mu.RUnlock()
		}
	}
}

// readPump 使用 gorilla/websocket API 重写
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		// ReadMessage 从连接中读取下一条消息
		_, rawMessage, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("用户 %d 的连接意外关闭: %v", c.UserID, err)
			} else {
				logrus.Infof("客户端 %d 从群组 %d 断开连接", c.UserID, c.GroupID)
			}
			break
		}

		// 创建一个临时的 WebSocketMsg 用于传递给 Service
		tempMsg := &dto.WebSocketMsg{
			UserID:  c.UserID,
			GroupID: c.GroupID,
			Data:    rawMessage, // 原始的、未解析的数据
		}

		// 持久化消息并获取干净的、可广播的消息体，以及需要通知的用户和通知内容
		cleanData, memberIDs, notificationData, err := c.Hub.ChatService.CreateMessage(c.Ctx, tempMsg)
		if err != nil {
			logrus.Errorf("处理用户 %d 的消息时出错: %v", c.UserID, err)

			// 定义一个错误消息结构体，用于通知客户端
			errorPayload := struct {
				Type    string `json:"type"`
				Content string `json:"content"`
			}{
				Type:    "error",
				Content: "消息发送失败，请重试。",
			}

			// 将错误消息序列化为 JSON
			errorJSON, marshalErr := json.Marshal(errorPayload)
			if marshalErr != nil {
				logrus.Errorf("严重错误: 序列化错误消息失败: %v", marshalErr)
				continue
			}

			// 将错误消息只发送给当前客户端
			select {
			case c.Send <- errorJSON:
			default:
				// 如果发送通道已满，客户端可能已断开连接或严重延迟
				logrus.Errorf("向客户端 %d 发送错误消息失败，发送通道已满。", c.UserID)
			}

			continue // 继续下一次循环
		}

		// 将处理后的干净消息广播给群组
		if cleanData != nil {
			// 为了生成正确的 JSON，我们创建一个临时的匿名 struct。
			finalMsgForMarshal := struct {
				UserID  int            `json:"user_id"`
				GroupID int            `json:"group_id"`
				Data    *dto.ClientMsg `json:"data"`
			}{
				UserID:  c.UserID,
				GroupID: c.GroupID,
				Data:    cleanData,
			}

			// 将这个临时结构体序列化为 JSON
			jsonData, err := json.Marshal(finalMsgForMarshal)
			if err != nil {
				logrus.Errorf("广播消息序列化失败: %v", err)
			} else {
				// 创建广播载荷
				payload := &BroadcastPayload{
					GroupID: c.GroupID,
					Data:    jsonData,
				}
				// 将载荷放入广播频道
				c.Hub.Broadcast <- payload
			}
		}

		// 发送个人通知
		if len(memberIDs) > 0 && notificationData != nil {
			for _, memberID := range memberIDs {
				// 不给消息发送者自己发通知
				if memberID == c.UserID {
					continue
				}
				c.Hub.DirectMessage <- &DirectMessagePayload{
					TargetUserID: memberID,
					Data:         notificationData,
				}
			}
		}
	}
}

// writePump 使用 gorilla/websocket API 重写
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了 Send 通道
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 使用 NextWriter 优化性能，特别是在有多个消息排队时
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logrus.Errorf("为用户 %d 获取写入器失败: %v", c.UserID, err)
				return
			}
			w.Write(message)

			// 将此轮中所有排队的消息一次性写入
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'}) // 可以用换行符分隔多条 JSON 消息
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				logrus.Errorf("为用户 %d 关闭写入器失败: %v", c.UserID, err)
				return
			}
		case <-ticker.C:
			// 定时发送 Ping 消息以保持连接并检测死连接
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.Errorf("向用户 %d 发送 Ping 失败: %v", c.UserID, err)
				return
			}
		case <-c.Ctx.Done():
			// 上下文被取消（例如服务器关闭）
			return
		}
	}
}
