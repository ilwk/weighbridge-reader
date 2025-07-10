package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
)

func main() {
	// 创建测试PDF文件
	testPDFPath := createTestPDF()
	defer os.Remove(testPDFPath)

	// 并发测试
	fmt.Println("开始并发打印测试...")

	var wg sync.WaitGroup
	results := make([]string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			jobID, err := uploadAndPrint(testPDFPath, fmt.Sprintf("test_%d.pdf", index))
			if err != nil {
				results[index] = fmt.Sprintf("任务 %d 失败: %v", index, err)
			} else {
				results[index] = fmt.Sprintf("任务 %d 成功，JobID: %s", index, jobID)
			}
		}(i)
	}

	wg.Wait()

	fmt.Println("\n测试结果:")
	for i, result := range results {
		fmt.Printf("任务 %d: %s\n", i, result)
	}

	// 查询队列状态
	fmt.Println("\n查询队列状态:")
	status := getQueueStatus()
	fmt.Printf("队列状态: %+v\n", status)
}

func createTestPDF() string {
	// 创建一个简单的测试PDF内容（这里只是示例）
	content := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj

3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 4 0 R
>>
endobj

4 0 obj
<<
/Length 44
>>
stream
BT
/F1 12 Tf
72 720 Td
(Test PDF Content) Tj
ET
endstream
endobj

xref
0 5
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
0000000204 00000 n 
trailer
<<
/Size 5
/Root 1 0 R
>>
startxref
297
%%EOF`

	tmpFile, err := os.CreateTemp("", "test_*.pdf")
	if err != nil {
		log.Fatal(err)
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		log.Fatal(err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}

func uploadAndPrint(filePath, filename string) (string, error) {
	// 创建multipart请求
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	writer.Close()

	// 发送请求
	resp, err := http.Post("http://localhost:8080/print", writer.FormDataContentType(), &buf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP错误: %s, 响应: %s", resp.Status, string(body))
	}

	// 解析响应获取job_id
	// 这里简化处理，实际应该解析JSON
	return string(body), nil
}

func getQueueStatus() map[string]interface{} {
	resp, err := http.Get("http://localhost:8080/print/queue")
	if err != nil {
		log.Printf("查询队列状态失败: %v", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		return nil
	}

	fmt.Printf("队列状态响应: %s\n", string(body))
	return nil
}
