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

// 状态缓存与判断
var (
	lastBroadcast time.Time
	lastStatus    string
	lastWeight    float64
	minInterval   = 100 * time.Millisecond
)

func shouldUpdate(status string, weight float64) bool {
	if status != lastStatus || abs(weight-lastWeight) > 0.05 {
		lastStatus = status
		lastWeight = weight
		return true
	}
	return false
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func parseWeightData(frame string) (status string, weight float64, unit string) {
	if !(strings.Contains(frame, ",GS") || strings.Contains(frame, ",NT")) {
		return "", 0, ""
	}
	parts := strings.SplitN(frame, " ", 2)
	if len(parts) != 2 {
		return "", 0, ""
	}
	prefix := strings.TrimSpace(parts[0])
	valueWithUnit := strings.TrimSpace(parts[1])
	switch prefix {
	case "ST,GS":
	case "ST,NT":
		status = "stable"
	case "US,GS":
	case "US,NT":
		status = "unstable"
	case "OV,GS":
	case "OV,NT":
		status = "overload"
	case "ZR,GS":
	case "ZR,NT":
		status = "zero"
	default:
		status = "unknown"
	}
	unit = ""
	weightStr := valueWithUnit
	if strings.HasSuffix(valueWithUnit, "kg") {
		unit = "kg"
		weightStr = strings.TrimSuffix(valueWithUnit, "kg")
	}
	weightStr = strings.TrimSpace(weightStr)
	_, err := fmt.Sscanf(weightStr, "%f", &weight)
	if err != nil {
		log.Println("数值解析失败:", err)
		return status, 0, unit
	}
	return status, weight, unit
}

func startSerialReader() {
	mode := &serial.Mode{BaudRate: config.BaudRate}
	port, err := serial.Open(config.SerialPort, mode)
	if err != nil {
		log.Fatal("串口打开失败:", err)
	}
	reader := bufio.NewReader(port)
	lastBroadcast = time.Now()
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("串口读取错误:", err)
			continue
		}
		cleaned := string(bytes.TrimSpace(line))
		status, weight, unit := parseWeightData(cleaned)
		if status == "" {
			continue
		}
		if time.Since(lastBroadcast) >= minInterval && shouldUpdate(status, weight) {
			lastBroadcast = time.Now()
			msg := fmt.Sprintf(`{"status":"%s","weight":%.2f,"unit":"%s"}`, status, weight, unit)
			log.Println("推送:", msg)
			broadcast(msg)
		}
	}
}

func main() {
	loadConfig()
	go startSerialReader()
	addr := fmt.Sprintf(":%d", config.WebsocketPort)
	http.HandleFunc("/ws", wsHandler)
	log.Printf("地磅读取服务已启动，运行在 http://localhost%s/ws\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
