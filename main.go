package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"reader/internal/config"
	"reader/internal/print"
	"reader/internal/serial"
	"reader/internal/ws"

	"github.com/gorilla/mux"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

func initLogger() {
	exePath, err := os.Executable()
	if err != nil {
		logrus.Fatalf("获取可执行文件路径失败: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	logDir := filepath.Join(exeDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.Mkdir(logDir, 0755)
	}
	
	// 创建文件输出
	fileHook := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, time.Now().Format("2006-01-02")+".log"),
		MaxSize:    20, // 单个日志文件最大20MB
		MaxBackups: 7,  // 最多保留7个备份
		MaxAge:     30, // 最多保留30天
		Compress:   false,
	}
	
	// 设置logrus输出到文件和控制台
	logrus.SetOutput(io.MultiWriter(os.Stdout, fileHook))
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	
	// 默认只记录INFO及以上级别到文件，但控制台显示所有
	logrus.SetLevel(logrus.InfoLevel)
}

// 模拟数据生成器
func startMockDataGenerator(ctx context.Context, cfg *config.Config, callback func(string)) {
	go func() {
		logrus.WithFields(logrus.Fields{
			"module":   "MOCK",
			"interval": time.Duration(cfg.BroadcastInterval) * time.Millisecond,
		}).Info("模拟数据生成器启动")
		
		ticker := time.NewTicker(time.Duration(cfg.BroadcastInterval) * time.Millisecond)
		defer ticker.Stop()
		
		msgIndex := 0
		for {
			select {
			case <-ctx.Done():
				logrus.WithField("module", "MOCK").Info("模拟数据生成器停止")
				return
			case <-ticker.C:
				if len(cfg.MockMessages) > 0 {
					msg := cfg.MockMessages[msgIndex%len(cfg.MockMessages)]
					logrus.WithFields(logrus.Fields{
						"module": "MOCK",
						"data":   msg.Message,
					}).Debug("生成模拟数据")
					callback(msg.Message)
					msgIndex++
				}
			}
		}
	}()
}

func main() {
	initLogger()
	cfg := config.LoadConfig()
	
	logrus.WithFields(logrus.Fields{
		"module": "MAIN",
	}).Info("地磅读取服务启动中...")
	
	var modeStr string
	if cfg.MockMode {
		modeStr = "模拟模式"
	} else {
		modeStr = "串口模式"
	}
	
	fmt.Printf("地磅读取服务启动中...\n")
	fmt.Printf("运行模式: %s\n", modeStr)
	if cfg.MockMode {
		fmt.Printf("配置信息 - WebSocket端口: %d, 模拟消息数: %d, 推送间隔: %dms\n", 
			cfg.WebsocketPort, len(cfg.MockMessages), cfg.BroadcastInterval)
	} else {
		fmt.Printf("配置信息 - 串口: %s, 波特率: %d, WebSocket端口: %d, 推送间隔: %dms\n", 
			cfg.SerialPort, cfg.BaudRate, cfg.WebsocketPort, cfg.BroadcastInterval)
	}
	
	hub := ws.NewHub()
	dataCallback := func(msg string) {
		logrus.WithFields(logrus.Fields{
			"module": "MAIN",
			"data":   msg,
		}).Debug("推送消息")
		hub.Broadcast(msg)
	}

	// 用于控制模拟数据生成器的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var manager *serial.SerialManager
	if cfg.MockMode {
		// 启动模拟数据生成器
		startMockDataGenerator(ctx, cfg, dataCallback)
		logrus.WithField("module", "MAIN").Info("模拟数据生成器已启动")
	} else {
		// 启动串口管理器
		manager = serial.NewSerialManager(cfg.SerialPort, cfg.BaudRate, 
			time.Duration(cfg.BroadcastInterval)*time.Millisecond, dataCallback)
		manager.Start()
		defer manager.Stop()
	}

	// 设置优雅关闭信号处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		logrus.WithField("module", "MAIN").Info("收到关闭信号，正在清理资源...")
		fmt.Println("收到关闭信号，正在清理资源...")
		
		if cfg.MockMode {
			cancel() // 停止模拟数据生成器
		} else if manager != nil {
			manager.Stop()
		}
		
		logrus.WithField("module", "MAIN").Info("资源清理完成，程序退出")
		fmt.Println("资源清理完成，程序退出")
		os.Exit(0)
	}()
	
	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)
	r := mux.NewRouter()
	r.HandleFunc("/ws", hub.HandleWS)
	r.HandleFunc("/print", print.PrintHandler).Methods(http.MethodPost, http.MethodOptions)

	r.Use(mux.CORSMethodMiddleware(r))
	
	logrus.WithFields(logrus.Fields{
		"module": "MAIN",
		"address": addr,
		"mode": modeStr,
	}).Info("地磅读取服务已启动")
	
	fmt.Printf("地磅读取服务已启动，运行在 http://localhost%s (%s)\n", addr, modeStr)
	logrus.Fatal(http.ListenAndServe(addr, r))
}
