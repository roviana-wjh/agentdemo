package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go-agent/flow/collaborative_writing"
)

// AutoPublishScheduler 自动发布调度器
type AutoPublishScheduler struct {
	interval time.Duration
	stopChan chan struct{}
	topics   []string
}

// 默认主题列表
var defaultTopics = []string{
	"Redis缓存优化实战指南",
	"MySQL索引优化技巧",
	"Docker容器化部署最佳实践",
	"Nginx反向代理配置详解",
	"Git分支管理策略",
	"RESTful API设计原则",
	"微服务架构模式详解",
	"JWT身份认证实战",
	"SQL注入防护技术",
	"高并发系统设计思路",
	"分布式事务解决方案",
	"消息队列应用场景",
	"ElasticSearch搜索优化",
	"MongoDB数据建模",
	"Go并发编程最佳实践",
	"JavaScript闭包深入解析",
	"Vue3组件设计模式",
	"React Hooks实战技巧",
	"TypeScript类型体操",
	"Webpack构建优化",
}

// NewAutoPublishScheduler 创建自动发布调度器
func NewAutoPublishScheduler() *AutoPublishScheduler {
	return &AutoPublishScheduler{
		stopChan: make(chan struct{}),
		topics:   defaultTopics,
	}
}

// Start 启动定时任务
// dailyTime格式: "09:00" (每天的某个时间点)
func (s *AutoPublishScheduler) Start(dailyTime string) error {
	// 解析时间
	targetTime, err := time.Parse("15:04", dailyTime)
	if err != nil {
		return fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	hour := targetTime.Hour()
	minute := targetTime.Minute()

	log.Printf("[AutoPublish] 定时任务已配置: 每天 %02d:%02d", hour, minute)

	// 启动定时循环
	go s.runDailyTask(hour, minute)

	return nil
}

// runDailyTask 每天定时执行任务
func (s *AutoPublishScheduler) runDailyTask(hour, minute int) {
	for {
		// 计算下次执行时间
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

		// 如果今天的时间已过，调度到明天
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}

		waitDuration := next.Sub(now)
		log.Printf("[AutoPublish] 下次执行时间: %s (等待 %.1f 小时)",
			next.Format("2006-01-02 15:04:05"), waitDuration.Hours())

		// 等待到执行时间
		select {
		case <-time.After(waitDuration):
			s.executePublish()
		case <-s.stopChan:
			log.Printf("[AutoPublish] 定时任务已停止")
			return
		}
	}
}

// Stop 停止定时任务
func (s *AutoPublishScheduler) Stop() {
	close(s.stopChan)
}

// executePublish 执行发布任务
func (s *AutoPublishScheduler) executePublish() {
	log.Printf("========================================")
	log.Printf("[AutoPublish] 开始执行自动发布任务")
	log.Printf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================")

	// 随机选择一个主题
	topic := s.getRandomTopic()
	log.Printf("[AutoPublish] 选中主题: %s", topic)

	// 获取CollaborativeWriting Graph
	graph, err := collaborative_writing.GetCollaborativeWritingGraph()
	if err != nil {
		log.Printf("[AutoPublish] ❌ 获取写作系统失败: %v", err)
		return
	}

	// 构建请求
	req := collaborative_writing.WritingRequest{
		Topic:       topic,
		AuthorID:    999, // 官方账号
		TargetWords: 2000,
	}

	// 执行写作
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	startTime := time.Now()
	result, err := graph.Invoke(ctx, req)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Printf("[AutoPublish] ❌ 文章生成失败: %v", err)
		s.logError(topic, err)
		return
	}

	if !result.Success {
		log.Printf("[AutoPublish] ❌ 文章生成失败: %s", result.Error)
		s.logError(topic, fmt.Errorf(result.Error))
		return
	}

	// 成功
	log.Printf("[AutoPublish] ✅ 文章生成成功")
	log.Printf("  主题: %s", topic)
	log.Printf("  字数: %d", result.WordCount)
	log.Printf("  得分: %.1f", result.ReviewScore)
	log.Printf("  耗时: %.1f秒", elapsed.Seconds())

	// 暂时注释掉自动保存，先让生成功能可用
	// TODO: 调试500错误后再启用
	/*
	// 发布文章
	articleID, err := s.publishArticle(topic, result.Content, 999)
	if err != nil {
		log.Printf("[AutoPublish] ❌ 文章发布失败: %v", err)
		s.logError(topic, err)
		return
	}
	log.Printf("[AutoPublish] ✅ 文章已发布，ID: %d", articleID)
	*/

	// 保存到本地文件
	filename := fmt.Sprintf("auto_generated_%s.md", time.Now().Format("20060102_150405"))
	if err := os.WriteFile(filename, []byte(result.Content), 0644); err != nil {
		log.Printf("[AutoPublish] ⚠️  保存文件失败: %v", err)
	} else {
		log.Printf("[AutoPublish] 💾 文章已保存: %s", filename)
	}

	// 记录日志
	s.logSuccess(topic, result, elapsed)

	log.Printf("========================================")
	log.Printf("[AutoPublish] 今日任务完成！")
	log.Printf("========================================")
}

// getRandomTopic 随机获取一个主题
func (s *AutoPublishScheduler) getRandomTopic() string {
	return s.topics[rand.Intn(len(s.topics))]
}

// logSuccess 记录成功日志
func (s *AutoPublishScheduler) logSuccess(topic string, result collaborative_writing.WritingResult, elapsed time.Duration) {
	// TODO: 写入日志文件或数据库
	log.Printf("[AutoPublish] 记录成功日志: %s", topic)
}

// logError 记录错误日志
func (s *AutoPublishScheduler) logError(topic string, err error) {
	// TODO: 写入错误日志
	log.Printf("[AutoPublish] 记录错误日志: %s, error: %v", topic, err)
}

// ManualTrigger 手动触发一次发布（用于测试）
func (s *AutoPublishScheduler) ManualTrigger() {
	log.Printf("[AutoPublish] 手动触发发布任务")
	go s.executePublish()
}

// publishArticle 发布文章到博客系统
func (s *AutoPublishScheduler) publishArticle(title, content string, authorID uint) (uint, error) {
	// 1. 创建草稿
	draftReq := map[string]interface{}{
		"author_id": authorID,
		"title":     title,
		"content":   content,
		"image":     "",
	}

	draftBody, err := json.Marshal(draftReq)
	if err != nil {
		return 0, fmt.Errorf("marshal draft request: %w", err)
	}

	// 调用内部工具API创建草稿
	resp, err := http.Post(
		"http://localhost:8080/internal/v1/create_article_draft",
		"application/json",
		bytes.NewReader(draftBody),
	)
	if err != nil {
		return 0, fmt.Errorf("create draft: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("create draft failed (status %d): %s", resp.StatusCode, string(body))
	}

	var draftResult struct {
		ID     uint   `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&draftResult); err != nil {
		return 0, fmt.Errorf("decode draft response: %w", err)
	}

	log.Printf("[AutoPublish] 草稿已创建，ID: %d", draftResult.ID)

	// 2. 发布文章（需要JWT token，这里先返回草稿ID）
	// TODO: 如果需要自动发布，需要添加内部token或使用系统账号

	return draftResult.ID, nil
}
