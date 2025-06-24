package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// 配置结构体
type Config struct {
	SerialPort    string `json:"serial_port"`
	BaudRate      int    `json:"baud_rate"`
	WebsocketPort int    `json:"websocket_port"`
	PrinterName   string `json:"printer_name"`
}

func LoadConfig() (Config, error) {
	var config Config
	data, err := os.ReadFile("config.json")
	if err != nil {
		return config, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("解析配置失败: %w", err)
	}
	return config, nil
}
