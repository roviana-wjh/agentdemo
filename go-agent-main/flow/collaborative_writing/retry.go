package collaborative_writing

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int           // 最大重试次数
	InitialDelay time.Duration // 初始延迟
	MaxDelay     time.Duration // 最大延迟
	Multiplier   float64       // 延迟倍数
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     10 * time.Second,
	Multiplier:   2.0,
}

// WithRetry 包装函数，添加重试逻辑
func WithRetry[T any](
	ctx context.Context,
	operation string,
	fn func(ctx context.Context) (T, error),
	config RetryConfig,
) (T, error) {
	var result T
	var lastErr error

	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[Retry] %s 第%d次重试（延迟%v）", operation, attempt, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return result, ctx.Err()
			}

			// 指数退避
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}

		result, lastErr = fn(ctx)
		if lastErr == nil {
			if attempt > 0 {
				log.Printf("[Retry] %s 重试成功", operation)
			}
			return result, nil
		}

		// 检查是否应该重试
		if !shouldRetry(lastErr) {
			log.Printf("[Retry] %s 错误不可重试: %v", operation, lastErr)
			return result, lastErr
		}

		log.Printf("[Retry] %s 失败: %v", operation, lastErr)
	}

	return result, fmt.Errorf("%s 失败，已重试%d次: %w", operation, config.MaxRetries, lastErr)
}

// shouldRetry 判断错误是否应该重试
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 网络错误可以重试
	if contains(errStr, "timeout") || contains(errStr, "connection refused") ||
		contains(errStr, "network") || contains(errStr, "EOF") {
		return true
	}

	// API限流可以重试
	if contains(errStr, "rate limit") || contains(errStr, "429") ||
		contains(errStr, "too many requests") {
		return true
	}

	// 服务暂时不可用可以重试
	if contains(errStr, "503") || contains(errStr, "502") ||
		contains(errStr, "service unavailable") {
		return true
	}

	// 其他错误不重试（如参数错误、认证错误等）
	return false
}

// contains 检查字符串是否包含子串（忽略大小写）
func contains(s, substr string) bool {
	return indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	// 简化实现：直接比较
	sLen := len(s)
	subLen := len(substr)
	if subLen > sLen {
		return -1
	}

	for i := 0; i <= sLen-subLen; i++ {
		match := true
		for j := 0; j < subLen; j++ {
			c1 := s[i+j]
			c2 := substr[j]
			// 忽略大小写
			if c1 >= 'A' && c1 <= 'Z' {
				c1 = c1 + 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 = c2 + 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// SafeAgentCall 安全的Agent调用，带超时和错误恢复
func SafeAgentCall[T any](
	ctx context.Context,
	agentName string,
	timeout time.Duration,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	var result T

	// 创建带超时的context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 使用channel接收结果
	type response struct {
		result T
		err    error
	}
	ch := make(chan response, 1)

	// 执行Agent调用
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SafeCall] %s panic恢复: %v", agentName, r)
				ch <- response{err: fmt.Errorf("agent panic: %v", r)}
			}
		}()

		res, err := fn(ctx)
		ch <- response{result: res, err: err}
	}()

	// 等待结果或超时
	select {
	case resp := <-ch:
		return resp.result, resp.err
	case <-ctx.Done():
		return result, fmt.Errorf("%s 超时（%v）", agentName, timeout)
	}
}
