package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"weightbridge-reader/cmd/reader"
	"weightbridge-reader/internal/config"
	"weightbridge-reader/internal/ws"
)

func main() {
	cfg, err := config.LoadConfig("config.json")

	if err != nil {
		log.Fatal(err)
	}

	dataChan := make(chan string)

	go func() {
		reader.SetSimulate(cfg.Simulate)
		err := reader.ReadWeightFromSerial(cfg.SerialPort, cfg.BaudRate, dataChan)
		if err != nil {
			log.Fatal(err)
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./web/static/index.html")
		if err != nil {
			log.Fatal(err)
		}

		var data = struct {
			WebsocketPort int
		}{
			WebsocketPort: cfg.WebsocketPort,
		}
		err = tmpl.Execute(w, data)

		if err != nil {
			log.Fatal(err)
		}
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.HandleWebSocket(w, r, dataChan)
	})

	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)
	fmt.Printf("地磅读取服务已启动，运行在 http://localhost%s/ws\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
