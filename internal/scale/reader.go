package scale

import (
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"
)

var simulate = true // 启用模拟

func Reader(portName string, baud int, dataChan chan<- string) error {
	if simulate {
		return simulateData(dataChan)
	}
	ports, err := serial.GetPortsList()
	println("端口列表:", &ports)
	if err != nil {
		return err
	}

	mode := serial.Mode{
		BaudRate: baud,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, &mode)

	if err != nil {
		return err
	}

	defer port.Close()

	buf := make([]byte, 100)
	for {
		n, err := port.Read(buf)
		if err != nil {
			log.Println("串口读取错误:", err)
			continue
		}
		if n > 0 {
			dataChan <- string(buf[:n])
		}
	}
}

// 模拟数据发送
func simulateData(dataChan chan<- string) error {
	log.Println("🔁 启用模拟串口数据模式")
	ticker := time.NewTicker(2 * time.Second)

	defer ticker.Stop()

	counter := 1
	for range ticker.C {
		fakeData := fmt.Sprintf("+123452")
		dataChan <- fakeData
		counter++
	}
	return nil
}
