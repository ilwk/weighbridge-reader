package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.bug.st/serial"
)

// 配置结构体
type Config struct {
	SerialPort    string `json:"serial_port"`
	BaudRate      int    `json:"baud_rate"`
	WebsocketPort int    `json:"websocket_port"`
}

var config Config

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("配置文件读取失败: %v", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("配置文件解析失败: %v", err)
	}
}

// WebSocket 管理
var upgrader = websocket.Upgrader{}
var clients = make(map[*websocket.Conn]bool)
var mutex = &sync.Mutex{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
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

func parseWeightData(frame string) (status string, weight float64, unit string) {
	// 例: ST,GS     123.45kg
	// 去除空格前先匹配前缀
	prefixes := map[string]string{
		"ST,GS": "stable",
		"US,GS": "unstable",
		"OL,GS": "overload",
		"ST,NT": "zero",
	}

	for key, val := range prefixes {
		if strings.HasPrefix(frame, key) {
			status = val
			// 去除前缀后是重量数据
			valueWithUnit := strings.TrimSpace(strings.TrimPrefix(frame, key))
			unit = ""
			if strings.HasSuffix(valueWithUnit, "kg") {
				unit = "kg"
				valueWithUnit = strings.TrimSuffix(valueWithUnit, "kg")
			}
			valueWithUnit = strings.TrimSpace(valueWithUnit)
			_, err := fmt.Sscanf(valueWithUnit, "%f", &weight)
			if err != nil {
				log.Println("数值解析失败:", err)
				return status, 0, unit
			}
			return status, weight, unit
		}
	}

	return "", 0, ""
}

var latestStableFrame string
var frameMutex sync.Mutex

func startSerialReader() {
	mode := &serial.Mode{BaudRate: config.BaudRate}
	port, err := serial.Open(config.SerialPort, mode)
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
			time.Sleep(1 * time.Second)
			frameMutex.Lock()
			if latestStableFrame != "" {
				log.Println("推送:", latestStableFrame)
				broadcast(latestStableFrame)
			}
			frameMutex.Unlock()
		}
	}()
}

func main() {
	loadConfig()
	go startSerialReader()
	addr := fmt.Sprintf(":%d", config.WebsocketPort)
	http.HandleFunc("/ws", wsHandler)
	log.Printf("地磅读取服务已启动，运行在 http://localhost%s/ws\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
