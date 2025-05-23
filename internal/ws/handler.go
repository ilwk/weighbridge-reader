package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func HandleWebSocket(w http.ResponseWriter, r *http.Request, dataChan <-chan string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		println("WebSocket 升级失败:", err)
		return
	}
	defer conn.Close()

	for data := range dataChan {
		err := conn.WriteMessage(websocket.TextMessage, []byte(data))
		if err != nil {
			println("WebSocket 写入失败:", err)
			break
		}
	}

}
