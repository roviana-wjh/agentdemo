package api

import (
	"context"
	"net/http"
	"time"

	"go-agent/flow/collaborative_writing"

	"github.com/gin-gonic/gin"
)

// CollaborativeWriting 多Agent协作写作接口
func CollaborativeWriting(c *gin.Context) {
	var req collaborative_writing.WritingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 获取协作写作图
	graph, err := collaborative_writing.GetCollaborativeWritingGraph()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "collaborative writing not ready: " + err.Error()})
		return
	}

	// 设置超时（5分钟）
	ctx, cancel := c.Request.Context(), func() {}
	if req.TargetWords > 3000 {
		// 长文章给更多时间
		ctx, cancel = context.WithTimeout(c.Request.Context(), 10*time.Minute)
	} else {
		ctx, cancel = context.WithTimeout(c.Request.Context(), 5*time.Minute)
	}
	defer cancel()

	// 执行协作写作
	result, err := graph.Invoke(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "writing failed: " + err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, result)
}

// CollaborativeWritingStream 流式输出协作写作（未来支持）
func CollaborativeWritingStream(c *gin.Context) {
	// TODO: Phase 2 实现流式输出
	c.JSON(http.StatusNotImplemented, gin.H{"error": "streaming not yet implemented"})
}
