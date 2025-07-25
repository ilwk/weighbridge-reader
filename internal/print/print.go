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
	go func() {
		for task := range printQueue {
			err := doPrintPDF(task.pdfContent, task.filename, task.printerName)
			task.resultChan <- err
			close(task.resultChan)
		}
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
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		err = os.Mkdir(historyDir, 0755)
		if err != nil {
			return "", fmt.Errorf("创建history目录失败: %w", err)
		}
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
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, content); err != nil {
		return "", err
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
