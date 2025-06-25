package serial

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"reader/internal/config"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.bug.st/serial"
)

// WebSocket 管理
var upgrader = websocket.Upgrader{}
var clients = make(map[*websocket.Conn]bool)
var mutex = &sync.Mutex{}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
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

var latestStableFrame string
var frameMutex sync.Mutex

func InitSerial() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("配置文件加载失败:", err)
	}
	mode := &serial.Mode{BaudRate: cfg.BaudRate}
	port, err := serial.Open(cfg.SerialPort, mode)
	if err != nil {
		log.Fatal("串口打开失败:", err)
	}
	reader := bufio.NewReader(port)

	// goroutine: 持续读取帧，缓存最新稳定帧
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				log.Println("串口读取错误:", err)
				continue
			}
			cleaned := string(bytes.TrimSpace(line))
			if cleaned == "" {
				continue
			}

			// 只记录稳定帧（你也可以改为记录最新任何帧）
			if strings.HasPrefix(cleaned, "ST,GS") || strings.HasPrefix(cleaned, "ST,NT") {
				frameMutex.Lock()
				latestStableFrame = cleaned
				frameMutex.Unlock()
			}
		}
	}()

	// goroutine: 每秒推送一次最新帧
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			frameMutex.Lock()
			if latestStableFrame != "" {
				log.Println("推送:", latestStableFrame)
				broadcast(latestStableFrame)
			}
			frameMutex.Unlock()
		}
	}()
}
