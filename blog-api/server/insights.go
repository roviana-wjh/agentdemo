package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AIArticleSummary(c *gin.Context) {
	var req struct {
		ArticleID uint   `json:"article_id"`
		SessionID string `json:"session_id,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ArticleID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "article_id is required"})
		return
	}
	proxyInsight(c, "/v1/blog/articles/summary/stream", req)
}

func AICommentAnalysis(c *gin.Context) {
	var req struct {
		ArticleID uint   `json:"article_id,omitempty"`
		Limit     int    `json:"limit,omitempty"`
		SessionID string `json:"session_id,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	proxyInsight(c, "/v1/blog/comments/analyze/stream", req)
}

func AISimpleAnalytics(c *gin.Context) {
	var req struct {
		Question  string `json:"question,omitempty"`
		SessionID string `json:"session_id,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	proxyInsight(c, "/v1/blog/analytics/stream", req)
}

func proxyInsight(c *gin.Context, path string, input any) {
	resp, err := defaultAgentClient.InsightStream(c.Request.Context(), path, input)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}
