package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"weightbridge-reader/internal/config"

	"github.com/gorilla/websocket"
	"go.bug.st/serial"
)

// ----------- WebSocket ----------
var upgrader = websocket.Upgrader{}
var clients = make(map[*websocket.Conn]bool)
var mutex = &sync.Mutex{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true } // 允许跨域
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket 升级失败:", err)
		return
	}
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()
	log.Println("新的WebSocket客户端连接")
}

// 广播到所有客户端
func broadcast(message string) {
	mutex.Lock()
	defer mutex.Unlock()
	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("WebSocket 写入错误:", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

// ----------- 串口读取 ----------
func startSerialReader() {
	cfg, err := config.LoadConfig("config.json")
	mode := &serial.Mode{
		BaudRate: cfg.BaudRate,
		DataBits: 8,
	}
	port, err := serial.Open(cfg.SerialPort, mode) // 替换为你的串口
	if err != nil {
		log.Fatal("串口打开失败:", err)
	}
	reader := bufio.NewReader(port)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("串口读取错误:", err)
			continue
		}

		cleaned := string(bytes.TrimSpace(line))
		status, weight := parseWeightData(cleaned)
		if status != "" {
			msg := fmt.Sprintf(`{"status":"%s", "weight":%.2f}`, status, weight)
			log.Println("推送:", msg)
			broadcast(msg)
		}
	}
}

// ----------- 数据解析 ----------
func parseWeightData(frame string) (status string, weight float64) {
	// 例：ST,GS     0.0kg
	if !(strings.Contains(frame, ",GS") || strings.Contains(frame, ",NT")) {
		return "", 0
	}
	parts := strings.SplitN(frame, " ", 2)
	if len(parts) != 2 {
		return "", 0
	}

	prefix := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), "kg")

	switch prefix {
	case "ST,GS":
		status = "stable"
	case "US,GS":
		status = "unstable"
	case "OL,GS":
		status = "overload"
	case "ST,NT":
		status = "zero"
	default:
		status = "unknown"
	}

	_, err := fmt.Sscanf(valueStr, "%f", &weight)
	if err != nil {
		log.Println("数值解析失败:", err)
		return status, 0
	}

	return status, weight
}

// ----------- 启动服务 ----------
func main() {

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	go startSerialReader()
	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)

	http.HandleFunc("/ws", wsHandler)
	fmt.Printf("地磅读取服务已启动，运行在 http://localhost%s/ws\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
