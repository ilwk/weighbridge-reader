package print

import (
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

// doPrintPDF 负责实际的打印逻辑
func doPrintPDF(pdfContent io.Reader, filename, printerName string) error {
	log.Printf("开始打印文件: %s, 打印机: %s", filename, printerName)

	// 保存文件内容到临时文件
	tmpPDFPath, err := saveTempPDF(pdfContent, filename)
	if err != nil {
		return fmt.Errorf("保存临时文件失败: %w", err)
	}
	// 创建执行PowerShell的命令
	args := []string{tmpPDFPath}
	if printerName != "" {
		// 用双引号包裹打印机名称，防止空格导致的问题
		args = append(args, fmt.Sprintf("\"%s\"", printerName))
	}

	cmd := exec.Command("./assets/PDFtoPrinterWin7.exe", args...)
	// 输出结果

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
