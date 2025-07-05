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

	go func() {
		defer func() {
			conn.Close()
			h.lock.Lock()
			delete(h.clients, conn)
			h.lock.Unlock()
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
