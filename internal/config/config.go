package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// 模拟消息配置
type MockMessage struct {
	Message string `json:"message"`
}

// 配置结构体
type Config struct {
	SerialPort       string        `json:"serial_port"`
	BaudRate         int           `json:"baud_rate"`
	WebsocketPort    int           `json:"websocket_port"`
	PrinterName      string        `json:"printer_name"`
	MockMode         bool          `json:"mock_mode"`
	MockMessages     []MockMessage `json:"mock_messages"`
	BroadcastInterval int          `json:"broadcast_interval"` // 毫秒
}

var defaultConfig = Config{
	SerialPort:       "COM1",
	BaudRate:         9600,
	WebsocketPort:    9900,
	PrinterName:      "",
	MockMode:         false,
	BroadcastInterval: 500,
	MockMessages: []MockMessage{
		{Message: "ST,GS,+000.000kg"},
		{Message: "ST,GS,+001.234kg"},
		{Message: "ST,GS,+002.567kg"},
		{Message: "ST,GS,+000.000kg"},
	},
}

var (
	instance *Config
	once     sync.Once
)

func LoadConfig() *Config {
	once.Do(func() {
		path, _ := os.Executable()
		dir := filepath.Dir(path)
		configPath := filepath.Join(dir, "config.json")

		data, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("读取配置文件失败: %v，使用默认配置", err)
			instance = &defaultConfig
			return
		}

		temp := defaultConfig
		if err := json.Unmarshal(data, &temp); err != nil {
			log.Printf("解析配置文件失败: %v", err)
			instance = &defaultConfig
			return
		}

		instance = &temp
	})
	return instance
}
