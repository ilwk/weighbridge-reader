package main

import (
	"log"
	"net/http"
	"weightbridge-ws/internal/scale"
	"weightbridge-ws/internal/ws"
)

func main() {
	dataChan := make(chan string)
	go func() {
		err := scale.Reader("COM3", 9600, dataChan)
		if err != nil {
			log.Fatal(err)
		}
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.HandleWebSocket(w, r, dataChan)
	})

	println("地磅读取服务已启动，请访问 http://localhost:8080/ws")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
