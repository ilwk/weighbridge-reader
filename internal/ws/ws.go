package ws

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients map[*websocket.Conn]chan string
	lock    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]chan string),
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"module": "WebSocket",
			"error":  err,
		}).Error("连接升级失败")
		return
	}

	ch := make(chan string, 10)
	h.lock.Lock()
	h.clients[conn] = ch
	clientCount := len(h.clients)
	h.lock.Unlock()

	logrus.WithFields(logrus.Fields{
		"module":      "WebSocket",
		"clientCount": clientCount,
	}).Info("新客户端连接")

	// 只用一个goroutine处理发送消息
	go func() {
		defer func() {
			h.lock.Lock()
			close(ch)
			delete(h.clients, conn)
			remainingCount := len(h.clients)
			h.lock.Unlock()

			if err := conn.Close(); err != nil {
				logrus.WithFields(logrus.Fields{
					"module": "WebSocket",
					"error":  err,
				}).Warn("关闭连接时出错")
			}
			logrus.WithFields(logrus.Fields{
				"module":         "WebSocket",
				"remainingCount": remainingCount,
			}).Info("客户端连接已断开")
		}()

		for msg := range ch {
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"module": "WebSocket",
					"error":  err,
				}).Error("写入消息失败")
				// 不直接返回，而是继续处理其他消息
				// 连接会在defer中正确关闭
				break
			}
		}
	}()
}

func (h *Hub) GetClientCount() int {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return len(h.clients)
}

func (h *Hub) Broadcast(msg string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if len(h.clients) == 0 {
		return // 没有客户端连接，直接返回
	}

	var closedClients []*websocket.Conn

	for conn, ch := range h.clients {
		select {
		case ch <- msg:
			// 成功发送
		default:
			// 缓冲区满或连接已关闭，标记为需要关闭的客户端
			closedClients = append(closedClients, conn)
		}
	}

	// 批量处理需要关闭的客户端，避免在锁内进行网络操作
	for _, conn := range closedClients {
		if ch, exists := h.clients[conn]; exists {
			logrus.WithField("module", "WebSocket").Warn("客户端连接异常，移除连接")
			close(ch)
			delete(h.clients, conn)
			// 异步关闭连接，避免阻塞
			go func(c *websocket.Conn) {
				if err := c.Close(); err != nil {
					logrus.WithFields(logrus.Fields{
						"module": "WebSocket",
						"error":  err,
					}).Warn("关闭异常客户端连接时出错")
				}
			}(conn)
		}
	}

	if len(closedClients) > 0 {
		logrus.WithFields(logrus.Fields{
			"module":       "WebSocket",
			"removedCount": len(closedClients),
			"currentCount": len(h.clients),
		}).Warn("移除异常客户端")
	}
}
