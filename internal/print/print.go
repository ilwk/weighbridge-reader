package print

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	once    sync.Once
	exePath string
	exeDir  string
	initErr error
)

type printTask struct {
	pdfContent  io.Reader
	filename    string
	printerName string
	resultChan  chan error
}

var (
	printQueue chan printTask
	queueOnce  sync.Once
)

func startPrintWorker() {
	printQueue = make(chan printTask, 100) // 队列长度可根据需求调整
	logrus.WithField("module", "Print").Info("启动打印队列处理器")

	go func() {
		for task := range printQueue {
			logrus.WithFields(logrus.Fields{
				"module":   "Print",
				"filename": task.filename,
			}).Info("处理打印任务")

			err := doPrintPDF(task.pdfContent, task.filename, task.printerName)
			task.resultChan <- err
			close(task.resultChan)

			if err != nil {
				logrus.WithFields(logrus.Fields{
					"module":   "Print",
					"filename": task.filename,
					"error":    err,
				}).Error("任务失败")
			} else {
				logrus.WithFields(logrus.Fields{
					"module":   "Print",
					"filename": task.filename,
				}).Info("任务完成")
			}
		}
		logrus.WithField("module", "Print").Info("打印队列处理器退出")
	}()
}

// PrintPDF 使用嵌入的打印工具打印指定 PDF 文件内容（队列版）
func PrintPDF(pdfContent io.Reader, filename, printerName string) error {
	queueOnce.Do(startPrintWorker)
	resultChan := make(chan error, 1)
	printQueue <- printTask{
		pdfContent:  pdfContent,
		filename:    filename,
		printerName: printerName,
		resultChan:  resultChan,
	}
	return <-resultChan
}

// SavePDFToHistory 保存PDF到history目录，重名自动累加
func SavePDFToHistory(content io.Reader, filename string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	historyDir := filepath.Join(exeDir, "history")

	// 确保history目录存在
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		if err = os.MkdirAll(historyDir, 0755); err != nil {
			return "", fmt.Errorf("创建history目录失败: %w", err)
		}
		log.Printf("[Print] 创建history目录: %s", historyDir)
	}

	// 处理重名
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	if ext == "" {
		ext = ".pdf"
	}

	historyPath := filepath.Join(historyDir, base)
	idx := 1
	for {
		if _, err := os.Stat(historyPath); os.IsNotExist(err) {
			break
		}
		historyPath = filepath.Join(historyDir, fmt.Sprintf("%s(%d)%s", name, idx, ext))
		idx++
	}

	out, err := os.Create(historyPath)
	if err != nil {
		return "", fmt.Errorf("创建历史文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, content); err != nil {
		return "", fmt.Errorf("写入历史文件失败: %w", err)
	}

	return historyPath, nil
}

// doPrintPDF 负责实际的打印逻辑
func doPrintPDF(pdfContent io.Reader, filename, printerName string) error {
	log.Printf("开始打印文件: %s, 打印机: %s", filename, printerName)

	// 先把内容全部读到内存
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, pdfContent); err != nil {
		return fmt.Errorf("读取PDF内容失败: %w", err)
	}

	// 保存一份历史记录
	_, _ = SavePDFToHistory(bytes.NewReader(buf.Bytes()), filename)

	// 用于打印的内容
	printReader := bytes.NewReader(buf.Bytes())

	// 保存文件内容到临时文件
	tmpPDFPath, err := saveTempPDF(printReader, filename)
	if err != nil {
		return fmt.Errorf("保存临时文件失败: %w", err)
	}

	// 获取可执行文件所在目录，拼接 assets 路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	printerExePath := filepath.Join(exeDir, "assets", "PDFtoPrinterWin7.exe")

	args := []string{tmpPDFPath}
	if printerName != "" {
		args = append(args, fmt.Sprintf("\"%s\"", printerName))
	}

	cmd := exec.Command(printerExePath, args...)

	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		log.Printf("打印失败: %s, 错误: %v", filename, err)
		return fmt.Errorf("打印执行失败: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("打印完成: %s, 耗时: %v", filename, duration)
	defer os.Remove(tmpPDFPath)
	return nil
}

// saveTempPDF 保存文件内容到临时目录，使用唯一文件名避免并发冲突
func saveTempPDF(content io.Reader, filename string) (string, error) {
	ext := filepath.Ext(filename)
	name := filepath.Base(filename)
	if ext == "" {
		name += ".pdf"
	}

	// 添加时间戳和随机数确保文件名唯一
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%d_%s", timestamp, name)
	tmpPath := filepath.Join(os.TempDir(), uniqueName)

	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, content); err != nil {
		return "", err
	}

	return tmpPath, nil
}
