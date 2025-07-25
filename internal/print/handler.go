package print

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reader/internal/config"
	"github.com/sirupsen/logrus"
)

// PrintHandler 是 HTTP 上传和打印的入口
func PrintHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	cfg := config.LoadConfig()

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logrus.WithFields(logrus.Fields{
			"module": "Print",
			"error":  err,
		}).Error("解析表单失败")
		http.Error(w, "解析表单失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	uploadedFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"module": "Print",
			"error":  err,
		}).Error("无法获取上传文件")
		http.Error(w, "无法获取上传文件: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer uploadedFile.Close()

	printerName := r.FormValue("printer")
	if printerName == "" {
		printerName = cfg.PrinterName
	}

	logrus.WithFields(logrus.Fields{
		"module":     "Print",
		"filename":   fileHeader.Filename,
		"printer":    printerName,
		"fileSize":   fileHeader.Size,
	}).Info("接收打印请求")

	// 直接使用新的 PrintPDF 函数，它会处理文件保存
	if err := PrintPDF(uploadedFile, fileHeader.Filename, printerName); err != nil {
		logrus.WithFields(logrus.Fields{
			"module":   "Print",
			"filename": fileHeader.Filename,
			"error":    err,
		}).Error("打印失败")
		http.Error(w, "打印失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("打印任务已完成: %s", fileHeader.Filename),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithFields(logrus.Fields{
			"module": "Print",
			"error":  err,
		}).Error("响应编码失败")
	}
}
