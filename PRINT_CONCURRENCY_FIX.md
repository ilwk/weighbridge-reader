# 打印服务并发问题修复

## 问题描述

用户反馈在并发打印时，只打印了最后一条，其他打印任务被忽略或覆盖。

## 问题原因分析

1. **临时文件路径冲突**：多个并发请求上传相同文件名的文件时，会覆盖同一个临时文件
2. **打印工具资源竞争**：多个打印任务同时使用同一个打印工具，可能导致资源冲突
3. **缺乏并发控制**：没有对并发打印任务进行有效的队列管理

## 修复方案

### 1. 临时文件唯一化

修改 `saveTempPDF` 函数，使用时间戳确保每个临时文件都有唯一的名称：

```go
// 添加时间戳和随机数确保文件名唯一
timestamp := time.Now().UnixNano()
uniqueName := fmt.Sprintf("%d_%s", timestamp, name)
tmpPath := filepath.Join(os.TempDir(), uniqueName)
```

### 2. 实现打印队列系统

创建了完整的打印队列管理系统：

- **PrintJob**：表示单个打印任务，包含状态跟踪
- **PrintQueue**：队列管理器，负责任务调度和执行
- **并发安全**：使用互斥锁确保队列操作的线程安全

### 3. 新增API端点

- `/print`：上传文件并添加到打印队列
- `/print/status?job_id=xxx`：查询特定任务状态
- `/print/queue`：查询整个队列状态

### 4. 增强日志记录

添加了详细的日志记录，便于调试和监控：

- 打印任务开始和完成时间
- 任务执行耗时统计
- 错误信息详细记录

## 使用方法

### 上传文件打印

```bash
curl -X POST http://localhost:8080/print \
  -F "file=@document.pdf" \
  -F "printer=HP_LaserJet"
```

响应示例：
```json
{
  "success": true,
  "message": "打印任务已添加到队列: document.pdf",
  "job_id": "job_1703123456789123456"
}
```

### 查询任务状态

```bash
curl "http://localhost:8080/print/status?job_id=job_1703123456789123456"
```

响应示例：
```json
{
  "job_id": "job_1703123456789123456",
  "status": "completed",
  "created_at": "2023-12-21T10:30:45Z",
  "printer": "HP_LaserJet",
  "file": "/tmp/1703123456789123456_document.pdf"
}
```

### 查询队列状态

```bash
curl http://localhost:8080/print/queue
```

响应示例：
```json
{
  "total": 10,
  "pending": 3,
  "printing": 1,
  "completed": 5,
  "failed": 1,
  "running": true
}
```

## 测试验证

运行测试脚本验证并发打印功能：

```bash
cd test
go run concurrent_print_test.go
```

## 技术改进

1. **队列管理**：使用FIFO队列确保打印任务按顺序执行
2. **状态跟踪**：每个任务都有完整的状态生命周期
3. **错误处理**：完善的错误捕获和报告机制
4. **资源管理**：自动清理临时文件，避免磁盘空间浪费
5. **超时控制**：设置打印任务超时，避免任务卡死

## 性能优化

- 使用goroutine异步执行打印任务
- 队列处理使用ticker定时检查，避免忙等待
- 临时文件使用defer自动清理
- 互斥锁粒度优化，减少锁竞争

## 监控建议

1. 定期检查队列状态，确保没有积压的任务
2. 监控打印失败率，及时处理异常
3. 关注临时文件清理情况，避免磁盘空间不足
4. 记录打印任务执行时间，优化性能瓶颈 