package print

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reader/internal/config"
)

// PrintHandler 是 HTTP 上传和打印的入口
func PrintHandler(w http.ResponseWriter, r *http.Request) {
	cfg := config.LoadConfig()

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "解析表单失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	uploadedFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "无法获取上传文件: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer uploadedFile.Close()

	tmpPDFPath, err := saveTempPDF(uploadedFile, fileHeader.Filename)
	if err != nil {
		http.Error(w, "保存文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpPDFPath)

	printerName := r.FormValue("printer")
	if printerName == "" {
		printerName = cfg.PrinterName
	}

	if err := PrintPDF(tmpPDFPath, printerName); err != nil {
		http.Error(w, "打印失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "打印成功: %s", fileHeader.Filename)
}

// saveTempPDF 保存上传的 PDF 文件到临时目录
func saveTempPDF(file multipart.File, filename string) (string, error) {
	ext := filepath.Ext(filename)
	name := filepath.Base(filename)
	if ext == "" {
		name += ".pdf"
	}
	tmpPath := filepath.Join(os.TempDir(), name)

	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	return tmpPath, nil
}
