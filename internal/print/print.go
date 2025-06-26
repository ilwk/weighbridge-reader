package print

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reader/internal/config"
)

func PrintHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, "加载配置失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "解析表单失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "无法获取上传文件: "+err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	printerName := r.FormValue("printer")

	if printerName == "" {
		printerName = cfg.PrinterName
	}

	// 保存为临时文件
	tmpFilePath, err := saveTempFile(file, header.Filename)
	if err != nil {
		http.Error(w, "保存文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFilePath) // 打印后删除

	args := []string{tmpFilePath}

	if printerName != "" {
		args = append(args, printerName)
	}

	fmt.Print("打印任务参数: ", args)

	cmd := exec.Command("./dll/PDFtoPrinterWin7.exe", args...)

	// 执行打印命令
	err = cmd.Run()
	if err != nil {
		log.Printf("Print command failed: %v\n", err)
		http.Error(w, "Print failed", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "打印任务发送成功: %s", tmpFilePath)
}

// 保存临时文件
func saveTempFile(file multipart.File, filename string) (string, error) {
	tmpPath := filepath.Join(os.TempDir(), filename)
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	return tmpPath, err
}
