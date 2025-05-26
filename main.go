package main

import (
	"fmt"
	"log"
	"net/http"
	"weightbridge-reader/cmd/reader"
	"weightbridge-reader/internal/config"
	"weightbridge-reader/internal/ws"

	"fyne.io/systray"
	"fyne.io/systray/example/icon"
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

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.HandleWebSocket(w, r, dataChan)
	})

	addr := fmt.Sprintf(":%d", cfg.WebsocketPort)
	fmt.Printf("地磅读取服务已启动，运行在 http://localhost%s/ws\n", addr)
	http.ListenAndServe(addr, nil)

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("地磅读取服务")
	systray.SetTooltip("地磅读取服务")
	mQuit := systray.AddMenuItem("退出", "退出程序")
	mOpen := systray.AddMenuItem("打开页面", "打开程序")
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-mOpen.ClickedCh:
				fmt.Println("打开页面")
				return
			}
		}
	}()

}

func onExit() {
	// TODO
}
