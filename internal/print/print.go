package print

import (
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

//go:embed assets/*
var embeddedFiles embed.FS

var (
	once    sync.Once
	exePath string
	exeDir  string
	initErr error
)

// PrintPDF 使用嵌入的打印工具打印指定 PDF 文件
func PrintPDF(pdfPath, printerName string) error {
	log.Printf("开始打印文件: %s, 打印机: %s", pdfPath, printerName)

	if err := preparePrinterFiles(); err != nil {
		return fmt.Errorf("初始化打印组件失败: %w", err)
	}

	args := []string{pdfPath}
	if printerName != "" {
		args = append(args, printerName)
	}

	cmd := exec.Command(exePath, args...)
	cmd.Dir = exeDir

	// 设置超时时间，避免打印任务卡死
	// cmd.Timeout = 30 * time.Second

	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		log.Printf("打印失败: %s, 错误: %v", pdfPath, err)
		return fmt.Errorf("打印执行失败: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("打印完成: %s, 耗时: %v", pdfPath, duration)

	return nil
}

func preparePrinterFiles() error {
	once.Do(func() {
		log.Println("初始化打印组件...")
		exePath, initErr = extractFile("PDFtoPrinterWin7.exe")
		if initErr != nil {
			log.Printf("提取打印工具失败: %v", initErr)
			return
		}
		exeDir = filepath.Dir(exePath)
		log.Printf("打印工具路径: %s", exePath)

		configTmpPath, err := extractFile("PDF-XChange Viewer Settings.dat")
		if err != nil {
			initErr = err
			log.Printf("提取配置文件失败: %v", err)
			return
		}
		targetPath := filepath.Join(exeDir, "PDF-XChange Viewer Settings.dat")
		_ = os.Rename(configTmpPath, targetPath)
		log.Println("打印组件初始化完成")
	})
	return initErr
}

// extractFile 将 embed 中的资源释放为临时文件
func extractFile(embeddedPath string) (string, error) {
	data, err := embeddedFiles.Open("assets/" + embeddedPath)
	if err != nil {
		return "", err
	}
	defer data.Close()

	tmpPath := filepath.Join(os.TempDir(), filepath.Base(embeddedPath))
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, data)
	if err != nil {
		return "", err
	}

	return tmpPath, nil
}
