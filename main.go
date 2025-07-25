package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"reader/internal/config"
	"reader/internal/print"
	"reader/internal/serial"
	"reader/internal/ws"

	"github.com/gorilla/mux"
	"github.com/natefinch/lumberjack"
)

func initLogger() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("获取可执行文件路径失败: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	logDir := filepath.Join(exeDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.Mkdir(logDir, 0755)
	}
	log.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(logDir, time.Now().Format("2006-01-02")+".log"),
		MaxSize:    20, // 单个日志文件最大20MB
		MaxBackups: 7,  // 最多保留7个备份
		MaxAge:     30, // 最多保留30天
		Compress:   false,
	})
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	initLogger()
	cfg := config.LoadConfig()
	hub := ws.NewHub()
	manager := serial.NewSerialManager(cfg.SerialPort, cfg.BaudRate, func(msg string) {
		log.Println("推送消息:", msg)
		hub.Broadcast(msg)
	})

	manager.Start()
	defer manager.Stop()
	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)
	r := mux.NewRouter()
	r.HandleFunc("/ws", hub.HandleWS)
	r.HandleFunc("/print", print.PrintHandler).Methods(http.MethodPost, http.MethodOptions)

	r.Use(mux.CORSMethodMiddleware(r))
	fmt.Printf("地磅读取服务已启动，运行在 http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
