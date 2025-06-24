package print

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func PrintHandler(w http.ResponseWriter, r *http.Request) {
	// 确保是POST请求
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析multipart表单，限制最大10MB文件
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// 获取表单中的文件和打印机名称
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	printerName := r.FormValue("printer")

	// 创建临时文件
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, handler.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFilePath) // 打印完成后删除临时文件

	// 将上传的文件内容复制到临时文件
	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// 构建SumatraPDF打印命令
	args := []string{"-print-to", printerName, tempFilePath}
	if printerName == "" {
		args = []string{"-print", tempFilePath} // 使用默认打印机
	}

	cmd := exec.Command("./lib/SumatraPDF.exe", args...)

	// 执行打印命令
	err = cmd.Run()
	if err != nil {
		log.Printf("Print command failed: %v\n", err)
		http.Error(w, "Print failed", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Print job submitted successfully for file: %s", handler.Filename)
}
