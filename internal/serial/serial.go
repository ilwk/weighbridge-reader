package serial

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"go.bug.st/serial"
)

type SerialManager struct {
	ctx         context.Context
	cancel      context.CancelFunc
	lastMessage atomic.Value
	mu          sync.Mutex
	port        serial.Port
	portName    string
	baudRate    int
	onMessage   func(string)
}

func NewSerialManager(port string, baud int, onMessage func(string)) *SerialManager {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &SerialManager{
		ctx:       ctx,
		cancel:    cancel,
		portName:  port,
		baudRate:  baud,
		onMessage: onMessage,
	}
	mgr.lastMessage.Store("")
	return mgr
}

func (s *SerialManager) Start() {
	go s.readLoop()
	go s.pushLoop()
}

func (s *SerialManager) Stop() {
	s.cancel()
	if s.port != nil {
		s.port.Close()
	}
}

func (s *SerialManager) readLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		mode := &serial.Mode{BaudRate: s.baudRate}
		port, err := serial.Open(s.portName, mode)
		if err != nil {
			log.Println("[Serial] 打开失败，5秒后重试:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		s.mu.Lock()
		s.port = port
		s.mu.Unlock()

		log.Println("[Serial] 打开成功:", s.portName)
		reader := bufio.NewReader(port)

		for {
			select {
			case <-s.ctx.Done():
				port.Close()
				return
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					log.Println("[Serial] 读取错误:", err)
					break // 重新打开串口
				}
				readData := string(bytes.TrimSpace(line))
				if readData == "" {
					continue
				}
				s.lastMessage.Store(readData)
			}
		}
	}
}

func (s *SerialManager) pushLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			val := s.lastMessage.Load()
			if msg, ok := val.(string); ok && msg != "" && s.onMessage != nil {
				s.onMessage(msg)
			}
		}
	}
}
