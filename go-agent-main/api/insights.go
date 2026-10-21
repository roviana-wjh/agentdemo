package api

import (
	"go-agent/flow"
	"io"
	"net/http"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

func ArticleSummaryStream(c *gin.Context) {
	if !validBlogInternalToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		ArticleID uint   `json:"article_id"`
		SessionID string `json:"session_id,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ArticleID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "article_id is required"})
		return
	}
	flows, err := flow.GetInsightFlows()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	streamInsight(c, req.RequestID, req.SessionID, func() (*schema.StreamReader[*schema.Message], error) {
		return flows.ArticleSummary.Stream(c.Request.Context(), req.ArticleID)
	})
}

func CommentAnalysisStream(c *gin.Context) {
	if !validBlogInternalToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
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
	flows, err := flow.GetInsightFlows()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	streamInsight(c, req.RequestID, req.SessionID, func() (*schema.StreamReader[*schema.Message], error) {
		return flows.CommentAnalysis.Stream(c.Request.Context(), req.ArticleID, req.Limit)
	})
}

func SimpleAnalyticsStream(c *gin.Context) {
	if !validBlogInternalToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Question  string `json:"question,omitempty"`
		SessionID string `json:"session_id,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	flows, err := flow.GetInsightFlows()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	streamInsight(c, req.RequestID, req.SessionID, func() (*schema.StreamReader[*schema.Message], error) {
		return flows.SimpleAnalytics.Stream(c.Request.Context(), req.Question)
	})
}

func streamInsight(c *gin.Context, requestID, sessionID string, open func() (*schema.StreamReader[*schema.Message], error)) {
	writer, err := NewSSEWriter(c, requestID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}
	writer.Start()
	stream, err := open()
	if err != nil {
		writer.Error("insight_failed", err.Error())
		return
	}
	defer stream.Close()
	for {
		message, recvErr := stream.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				writer.Done()
				return
			}
			writer.Error("insight_stream_read_failed", recvErr.Error())
			return
		}
		if message != nil && message.Content != "" {
			writer.Delta(message.Content)
		}
	}
}
