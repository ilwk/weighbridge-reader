package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	SerialPort    string `json:"serial_port"`
	BaudRate      int    `json:"baud_rate"`
	WebsocketPort int    `json:"websocket_port"`
	Simulate      bool   `json:"simulate"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %v", err)
	}
	defer file.Close()

	decode := json.NewDecoder(file)
	var cfg Config
	if err := decode.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("配置文件格式错误: %v", err)
	}
	return &cfg, nil
}
