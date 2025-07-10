package print

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// PrintJob 表示一个打印任务
type PrintJob struct {
	ID          string
	PDFPath     string
	PrinterName string
	CreatedAt   time.Time
	Status      string // "pending", "printing", "completed", "failed"
	Error       error
}

// PrintQueue 打印队列管理器
type PrintQueue struct {
	jobs    []*PrintJob
	mutex   sync.RWMutex
	running bool
	stopCh  chan struct{}
}

var (
	queue     *PrintQueue
	queueOnce sync.Once
)

// GetPrintQueue 获取打印队列单例
func GetPrintQueue() *PrintQueue {
	queueOnce.Do(func() {
		queue = &PrintQueue{
			jobs:   make([]*PrintJob, 0),
			stopCh: make(chan struct{}),
		}
		go queue.processJobs()
	})
	return queue
}

// AddJob 添加打印任务到队列
func (pq *PrintQueue) AddJob(pdfPath, printerName string) string {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	job := &PrintJob{
		ID:          fmt.Sprintf("job_%d", time.Now().UnixNano()),
		PDFPath:     pdfPath,
		PrinterName: printerName,
		CreatedAt:   time.Now(),
		Status:      "pending",
	}

	pq.jobs = append(pq.jobs, job)
	log.Printf("添加打印任务: %s, 文件: %s", job.ID, pdfPath)

	return job.ID
}

// GetJobStatus 获取任务状态
func (pq *PrintQueue) GetJobStatus(jobID string) (*PrintJob, bool) {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()

	for _, job := range pq.jobs {
		if job.ID == jobID {
			return job, true
		}
	}
	return nil, false
}

// processJobs 处理打印队列
func (pq *PrintQueue) processJobs() {
	pq.running = true
	defer func() { pq.running = false }()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pq.stopCh:
			return
		case <-ticker.C:
			pq.processNextJob()
		}
	}
}

// processNextJob 处理下一个打印任务
func (pq *PrintQueue) processNextJob() {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	// 查找待处理的任务
	var pendingJob *PrintJob
	for _, job := range pq.jobs {
		if job.Status == "pending" {
			pendingJob = job
			break
		}
	}

	if pendingJob == nil {
		return
	}

	// 标记为正在打印
	pendingJob.Status = "printing"
	log.Printf("开始处理打印任务: %s", pendingJob.ID)

	// 在goroutine中执行打印，避免阻塞队列处理
	go func(job *PrintJob) {
		if err := PrintPDF(job.PDFPath, job.PrinterName); err != nil {
			job.Status = "failed"
			job.Error = err
			log.Printf("打印任务失败: %s, 错误: %v", job.ID, err)
		} else {
			job.Status = "completed"
			log.Printf("打印任务完成: %s", job.ID)
		}
	}(pendingJob)
}

// Stop 停止打印队列
func (pq *PrintQueue) Stop() {
	close(pq.stopCh)
}

// GetQueueStatus 获取队列状态
func (pq *PrintQueue) GetQueueStatus() map[string]interface{} {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()

	pending := 0
	printing := 0
	completed := 0
	failed := 0

	for _, job := range pq.jobs {
		switch job.Status {
		case "pending":
			pending++
		case "printing":
			printing++
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}

	return map[string]interface{}{
		"total":     len(pq.jobs),
		"pending":   pending,
		"printing":  printing,
		"completed": completed,
		"failed":    failed,
		"running":   pq.running,
	}
}
