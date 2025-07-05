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
	if err := preparePrinterFiles(); err != nil {
		return fmt.Errorf("初始化打印组件失败: %w", err)
	}

	args := []string{pdfPath}
	if printerName != "" {
		args = append(args, printerName)
	}

	cmd := exec.Command(exePath, args...)
	cmd.Dir = exeDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("打印执行失败: %w", err)
	}

	return nil
}

func preparePrinterFiles() error {
	once.Do(func() {
		exePath, initErr = extractFile("PDFtoPrinterWin7.exe")
		if initErr != nil {
			return
		}
		exeDir = filepath.Dir(exePath)

		configTmpPath, err := extractFile("PDF-XChange Viewer Settings.dat")
		if err != nil {
			initErr = err
			return
		}
		targetPath := filepath.Join(exeDir, "PDF-XChange Viewer Settings.dat")
		_ = os.Rename(configTmpPath, targetPath)
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
	log.Printf("释放 %s", os.TempDir())
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
