package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

type MessageConfig struct {
	Interval int    `json:"interval"`         // 毫秒
	Message  string `json:"message"`          // 消息内容
	Repeat   int    `json:"repeat,omitempty"` // 推送次数（0=无限）
}

type Config struct {
	Port     int             `json:"port"`
	Messages []MessageConfig `json:"messages"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleConnection(conn *websocket.Conn, messages []MessageConfig) {
	defer conn.Close()
	log.Println("客户端已连接")

	for {
		for _, msg := range messages {
			times := msg.Repeat
			if times == 0 {
				// 无限重复此消息
				for {
					if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Message)); err != nil {
						log.Println("发送失败:", err)
						return
					}
					time.Sleep(time.Duration(msg.Interval) * time.Millisecond)
				}
			} else {
				// 重复指定次数
				for i := 0; i < times; i++ {
					if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Message)); err != nil {
						log.Println("发送失败:", err)
						return
					}
					time.Sleep(time.Duration(msg.Interval) * time.Millisecond)
				}
			}
		}
	}
}

func startWebSocketServer(config Config) {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket 升级失败:", err)
			return
		}
		go handleConnection(conn, config.Messages)
	})

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("WebSocket 服务运行于: ws://localhost%s/ws\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadConfigFromFile(path string) (Config, error) {
	currentPath, _ := os.Executable()
	dir := filepath.Dir(currentPath)
	configPath := filepath.Join(dir, path)
	var config Config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("解析配置失败: %w", err)
	}
	return config, nil
}

func main() {
	config, err := loadConfigFromFile("./config.json")
	if err != nil {
		log.Println(err)
	}
	startWebSocketServer(config)
}
