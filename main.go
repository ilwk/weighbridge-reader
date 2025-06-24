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
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	// go serial.InitSerial()
	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)
	http.HandleFunc("/ws", serial.HandleWebSocket)
	http.HandleFunc("/print", print.PrintHandler)
	log.Printf("地磅读取服务已启动，运行在 http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
