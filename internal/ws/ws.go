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
		log.Println("[WebSocket] 升级失败:", err)
		return
	}

	ch := make(chan string, 10)
	h.lock.Lock()
	h.clients[conn] = ch
	h.lock.Unlock()

	log.Println("[WebSocket] 新客户端连接")

	// 只用一个goroutine处理发送消息
	go func() {
		defer func() {
			h.lock.Lock()
			close(ch)
			delete(h.clients, conn)
			h.lock.Unlock()
			conn.Close()
			log.Println("[WebSocket] 客户端连接已断开")
		}()

		for msg := range ch {
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Println("[WebSocket] 写入失败:", err)
				return
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
	
	for conn, ch := range h.clients {
		select {
		case ch <- msg:
			// 成功发送
		default:
			// 缓冲区满，移除慢客户端
			log.Println("[WebSocket] 推送缓冲满，移除慢客户端")
			close(ch)
			delete(h.clients, conn)
			conn.Close()
		}
	}
}
