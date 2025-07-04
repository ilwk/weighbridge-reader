package main

import (
	"fmt"
	"log"
	"net/http"

	"reader/internal/config"
	"reader/internal/print"
	"reader/internal/serial"
)

func main() {
	cfg := config.LoadConfig()
	go serial.InitSerial()
	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", serial.HandleWebSocket)
	mux.HandleFunc("/print", print.PrintHandler)
	handler := withCORS(mux)
	log.Printf("地磅读取服务已启动，运行在 http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

// CORS 中间件：允许所有跨域请求
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置跨域响应头
		w.Header().Set("Access-Control-Allow-Origin", "*") // 生产建议改为指定域名
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}
