package serial

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"go.bug.st/serial"
)

type SerialManager struct {
	ctx               context.Context
	cancel            context.CancelFunc
	lastMessage       atomic.Value
	mu                sync.Mutex
	port              serial.Port
	portName          string
	baudRate          int
	onMessage         func(string)
	retryCount        int
	maxRetries        int
	retryInterval     time.Duration
	broadcastInterval time.Duration
}

func NewSerialManager(port string, baud int, broadcastInterval time.Duration, onMessage func(string)) *SerialManager {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &SerialManager{
		ctx:               ctx,
		cancel:            cancel,
		portName:          port,
		baudRate:          baud,
		onMessage:         onMessage,
		retryCount:        0,
		maxRetries:        10,
		retryInterval:     5 * time.Second,
		broadcastInterval: broadcastInterval,
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
			logrus.WithField("module", "Serial").Info("接收到停止信号，退出读取循环")
			return
		default:
		}

		port, err := s.openPortWithRetry()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"module": "Serial",
				"error":  err,
			}).Error("端口打开失败，达到最大重试次数")
			time.Sleep(30 * time.Second) // 长时间等待后重试
			s.retryCount = 0 // 重置重试计数
			continue
		}

		s.mu.Lock()
		s.port = port
		s.mu.Unlock()

		logrus.WithFields(logrus.Fields{
			"module":   "Serial",
			"port":     s.portName,
			"baudRate": s.baudRate,
		}).Info("端口打开成功")
		s.retryCount = 0 // 成功后重置重试计数

		reader := bufio.NewReader(port)
		for {
			select {
			case <-s.ctx.Done():
				port.Close()
				logrus.WithFields(logrus.Fields{
					"module": "Serial",
					"port":   s.portName,
				}).Info("关闭端口")
				return
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"module": "Serial",
						"error":  err,
					}).Error("读取错误，重新打开端口")
					port.Close()
					break // 重新打开串口
				}
				readData := string(bytes.TrimSpace(line))
				if readData == "" {
					continue
				}
				if strings.HasPrefix(readData, "ST,GS") {
					s.lastMessage.Store(readData)
					// 数据接收用Debug级别，不会输出到文件日志
					logrus.WithFields(logrus.Fields{
						"module": "Serial",
						"data":   readData,
					}).Debug("接收数据")
				}
			}
		}
	}
}

// openPortWithRetry 尝试打开串口，带有退避重试机制
func (s *SerialManager) openPortWithRetry() (serial.Port, error) {
	mode := &serial.Mode{BaudRate: s.baudRate}
	
	for s.retryCount < s.maxRetries {
		select {
		case <-s.ctx.Done():
			return nil, fmt.Errorf("操作已取消")
		default:
		}

		port, err := serial.Open(s.portName, mode)
		if err == nil {
			return port, nil
		}

		s.retryCount++
		// 指数退避：重试间隔逐渐增加
		retryDelay := s.retryInterval * time.Duration(s.retryCount)
		if retryDelay > 30*time.Second {
			retryDelay = 30 * time.Second
		}

		logrus.WithFields(logrus.Fields{
			"module":     "Serial",
			"attempt":    s.retryCount,
			"maxRetries": s.maxRetries,
			"error":      err,
			"retryDelay": retryDelay,
		}).Error("打开端口失败")
		
		time.Sleep(retryDelay)
	}

	return nil, fmt.Errorf("达到最大重试次数 %d，最后错误: 无法打开串口 %s", s.maxRetries, s.portName)
}

func (s *SerialManager) pushLoop() {
	ticker := time.NewTicker(s.broadcastInterval)
	defer ticker.Stop()
	logrus.WithFields(logrus.Fields{
		"module":   "Serial",
		"interval": s.broadcastInterval,
	}).Info("启动数据推送循环")
	
	for {
		select {
		case <-s.ctx.Done():
			logrus.WithField("module", "Serial").Info("数据推送循环退出")
			return
		case <-ticker.C:
			val := s.lastMessage.Load()
			if msg, ok := val.(string); ok && msg != "" && s.onMessage != nil {
				s.onMessage(msg)
			}
		}
	}
}
