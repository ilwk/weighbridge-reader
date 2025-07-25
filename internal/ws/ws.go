package ws

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients map[*websocket.Conn]chan string
	lock    sync.Mutex
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
		log.Println("[WebSocket] 升级失败:", err)
		return
	}
	ch := make(chan string, 10)
	h.lock.Lock()
	h.clients[conn] = ch
	h.lock.Unlock()

	log.Println("[WebSocket] 新客户端连接")

	// 写消息 goroutine
	go func() {
		defer func() {
			conn.Close()
			h.lock.Lock()
			delete(h.clients, conn)
			h.lock.Unlock()
			close(ch)
			log.Println("[WebSocket] 客户端断开，资源已清理")
		}()
		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				log.Println("[WebSocket] 写入失败:", err)
				break // 触发 defer，清理资源
			}
		}
	}()

	// 读消息 goroutine，及时发现客户端关闭
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				log.Println("[WebSocket] 读取失败/客户端关闭:", err)
				// 主动关闭写入通道，触发写 goroutine 退出
				h.lock.Lock()
				if _, ok := h.clients[conn]; ok {
					delete(h.clients, conn)
					close(ch)
				}
				h.lock.Unlock()
				conn.Close()
				break
			}
		}
	}()
}

func (h *Hub) Broadcast(msg string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	for _, ch := range h.clients {
		select {
		case ch <- msg:
		default:
			log.Println("[WebSocket] 推送缓冲满，跳过")
		}
	}
}
