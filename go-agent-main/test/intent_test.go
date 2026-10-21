package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// TestCase 测试用例结构
type TestCase struct {
	ID                 string  `json:"id"`
	Query              string  `json:"query"`
	ExpectedIntent     string  `json:"expected_intent"`
	ExpectedConfidence string  `json:"expected_confidence"`
	Reason             string  `json:"reason"`
}

// TestCategory 测试分类
type TestCategory struct {
	Category string     `json:"category"`
	Cases    []TestCase `json:"cases"`
}

// TestDataset 测试数据集
type TestDataset struct {
	TestCases []TestCategory `json:"test_cases"`
}

// TestResult 测试结果
type TestResult struct {
	ID                string  `json:"id"`
	Query             string  `json:"query"`
	ExpectedIntent    string  `json:"expected_intent"`
	ActualIntent      string  `json:"actual_intent"`
	ExpectedConfStr   string  `json:"expected_confidence"`
	ActualConfidence  float64 `json:"actual_confidence"`
	ClassifyMethod    string  `json:"classify_method"`
	Passed            bool    `json:"passed"`
	ResponseTime      int64   `json:"response_time_ms"`
	Error             string  `json:"error,omitempty"`
}

// SSEEvent SSE事件
type SSEEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// RouteData 路由事件数据
type RouteData struct {
	Intent     string  `json:"intent"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

func main() {
	// 读取测试数据集
	dataset, err := loadTestDataset("test/intent_test_dataset.json")
	if err != nil {
		fmt.Printf("❌ 加载测试数据集失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🧪 开始意图识别测试")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("📊 测试用例总数: %d\n", countTotalCases(dataset))
	fmt.Println()

	// 执行测试
	results := runTests(dataset, "http://localhost:8081")

	// 生成报告
	generateReport(results)
}

func loadTestDataset(path string) (*TestDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var dataset TestDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, err
	}

	return &dataset, nil
}

func countTotalCases(dataset *TestDataset) int {
	count := 0
	for _, category := range dataset.TestCases {
		count += len(category.Cases)
	}
	return count
}

func runTests(dataset *TestDataset, baseURL string) []TestResult {
	var results []TestResult
	totalCases := 0
	passedCases := 0

	for _, category := range dataset.TestCases {
		fmt.Printf("📁 测试类别: %s\n", category.Category)
		fmt.Println(strings.Repeat("-", 60))

		for _, testCase := range category.Cases {
			totalCases++
			result := runSingleTest(baseURL, testCase)
			results = append(results, result)

			// 打印测试结果
			status := "❌"
			if result.Passed {
				status = "✅"
				passedCases++
			}

			fmt.Printf("%s [%s] %s\n", status, result.ID, result.Query)
			fmt.Printf("   预期: %s | 实际: %s (%.2f) | 方法: %s | 耗时: %dms\n",
				result.ExpectedIntent,
				result.ActualIntent,
				result.ActualConfidence,
				result.ClassifyMethod,
				result.ResponseTime)

			if !result.Passed {
				fmt.Printf("   ⚠️  失败原因: %s\n", result.Error)
			}
			fmt.Println()

			// 避免请求过快
			time.Sleep(100 * time.Millisecond)
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("🎯 测试完成: %d/%d 通过 (%.1f%%)\n",
		passedCases, totalCases, float64(passedCases)/float64(totalCases)*100)
	fmt.Println()

	return results
}

func runSingleTest(baseURL string, testCase TestCase) TestResult {
	result := TestResult{
		ID:              testCase.ID,
		Query:           testCase.Query,
		ExpectedIntent:  testCase.ExpectedIntent,
		ExpectedConfStr: testCase.ExpectedConfidence,
	}

	startTime := time.Now()

	// 构建请求
	reqBody := map[string]string{
		"query":      testCase.Query,
		"session_id": "test_session_" + testCase.ID,
	}
	reqData, _ := json.Marshal(reqBody)

	// 发送请求
	resp, err := http.Post(
		baseURL+"/v1/unified/chat/stream",
		"application/json",
		bytes.NewBuffer(reqData),
	)

	if err != nil {
		result.Error = fmt.Sprintf("请求失败: %v", err)
		result.ResponseTime = time.Since(startTime).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		result.ResponseTime = time.Since(startTime).Milliseconds()
		return result
	}

	// 解析SSE流，提取route事件
	routeFound := false
	scanner := NewSSEScanner(resp.Body)

	for scanner.Scan() {
		event, data := scanner.Event()

		if event == "route" {
			var routeData RouteData
			if err := json.Unmarshal(data, &routeData); err == nil {
				result.ActualIntent = routeData.Intent
				result.ActualConfidence = routeData.Confidence
				result.ClassifyMethod = routeData.Reason
				routeFound = true
				break
			}
		}

		if event == "error" {
			var errData map[string]string
			json.Unmarshal(data, &errData)
			result.Error = errData["message"]
			break
		}
	}

	result.ResponseTime = time.Since(startTime).Milliseconds()

	if !routeFound {
		result.Error = "未找到route事件"
		return result
	}

	// 验证结果
	result.Passed = validateResult(result)

	return result
}

func validateResult(result TestResult) bool {
	// 验证意图是否正确
	if result.ActualIntent != result.ExpectedIntent {
		result.Error = fmt.Sprintf("意图不匹配: 预期 %s, 实际 %s",
			result.ExpectedIntent, result.ActualIntent)
		return false
	}

	// 验证置信度是否符合预期范围
	confStr := result.ExpectedConfStr
	if strings.HasPrefix(confStr, ">") {
		threshold := parseFloat(strings.TrimPrefix(confStr, ">"))
		if result.ActualConfidence <= threshold {
			result.Error = fmt.Sprintf("置信度过低: 预期 >%.2f, 实际 %.2f",
				threshold, result.ActualConfidence)
			return false
		}
	} else if strings.Contains(confStr, "-") {
		parts := strings.Split(confStr, "-")
		min := parseFloat(parts[0])
		max := parseFloat(parts[1])
		if result.ActualConfidence < min || result.ActualConfidence > max {
			result.Error = fmt.Sprintf("置信度超出范围: 预期 %.2f-%.2f, 实际 %.2f",
				min, max, result.ActualConfidence)
			return false
		}
	}

	return true
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func generateReport(results []TestResult) {
	// 统计数据
	total := len(results)
	passed := 0
	failedByIntent := make(map[string]int)
	failedByMethod := make(map[string]int)
	avgResponseTime := int64(0)

	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failedByIntent[r.ExpectedIntent]++
			failedByMethod[r.ClassifyMethod]++
		}
		avgResponseTime += r.ResponseTime
	}
	avgResponseTime /= int64(total)

	accuracy := float64(passed) / float64(total) * 100

	// 生成报告文件
	reportFile := "test/intent_test_report.json"
	report := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"summary": map[string]interface{}{
			"total_cases":         total,
			"passed":              passed,
			"failed":              total - passed,
			"accuracy":            accuracy,
			"avg_response_time":   avgResponseTime,
			"failed_by_intent":    failedByIntent,
			"failed_by_method":    failedByMethod,
		},
		"details": results,
	}

	reportData, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(reportFile, reportData, 0644)

	fmt.Printf("📄 详细报告已保存: %s\n", reportFile)
	fmt.Printf("⚡ 平均响应时间: %dms\n", avgResponseTime)
}

// SSEScanner SSE流扫描器
type SSEScanner struct {
	reader *io.Reader
	buffer []byte
}

func NewSSEScanner(r io.Reader) *SSEScanner {
	return &SSEScanner{reader: &r}
}

func (s *SSEScanner) Scan() bool {
	// 简化实现：逐行读取
	buf := make([]byte, 4096)
	n, err := (*s.reader).Read(buf)
	if err != nil || n == 0 {
		return false
	}
	s.buffer = buf[:n]
	return true
}

func (s *SSEScanner) Event() (string, []byte) {
	lines := strings.Split(string(s.buffer), "\n")
	var event string
	var data []byte

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	return event, data
}
