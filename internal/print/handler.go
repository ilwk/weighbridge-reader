package print

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reader/internal/config"
	"sync"
	"time"
)

var printMutex sync.Mutex

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

	// 使用打印队列处理打印任务
	queue := GetPrintQueue()
	jobID := queue.AddJob(tmpPDFPath, printerName)

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("打印任务已添加到队列: %s", fileHeader.Filename),
		"job_id":  jobID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StatusHandler 查询打印任务状态
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "缺少job_id参数", http.StatusBadRequest)
		return
	}

	queue := GetPrintQueue()
	job, exists := queue.GetJobStatus(jobID)

	if !exists {
		http.Error(w, "任务不存在", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"job_id":     job.ID,
		"status":     job.Status,
		"created_at": job.CreatedAt,
		"printer":    job.PrinterName,
		"file":       job.PDFPath,
	}

	if job.Error != nil {
		response["error"] = job.Error.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// QueueStatusHandler 查询打印队列状态
func QueueStatusHandler(w http.ResponseWriter, r *http.Request) {
	queue := GetPrintQueue()
	status := queue.GetQueueStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// saveTempPDF 保存上传的 PDF 文件到临时目录，使用唯一文件名避免并发冲突
func saveTempPDF(file multipart.File, filename string) (string, error) {
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

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	return tmpPath, nil
}
