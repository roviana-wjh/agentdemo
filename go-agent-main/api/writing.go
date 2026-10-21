package api

import (
	"crypto/subtle"
	"go-agent/config"
	"go-agent/flow"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func WritingStream(c *gin.Context) {
	if !validBlogInternalToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Topic        string `json:"topic"`
		Requirements string `json:"requirements,omitempty"`
		SessionID    string `json:"session_id,omitempty"`
		RequestID    string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Topic) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
		return
	}
	writer, err := NewSSEWriter(c, req.RequestID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}
	writer.Start()
	writing, err := flow.GetWritingFlow()
	if err != nil {
		writer.Error("writing_not_ready", err.Error())
		return
	}
	stream, err := writing.Stream(c.Request.Context(), flow.WritingRequest{Topic: req.Topic, Requirements: req.Requirements})
	if err != nil {
		writer.Error("writing_failed", err.Error())
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
			writer.Error("writing_stream_read_failed", recvErr.Error())
			return
		}
		if message != nil && message.Content != "" {
			writer.Delta(message.Content)
		}
	}
}

func SaveWritingDraft(c *gin.Context) {
	if !validBlogInternalToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req flow.SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	writing, err := flow.GetWritingFlow()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	result, err := writing.SaveDraft(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func validBlogInternalToken(c *gin.Context) bool {
	if config.Cfg == nil {
		return false
	}
	expected := strings.TrimSpace(config.Cfg.BlogAPIConf.InternalToken)
	provided := strings.TrimSpace(c.GetHeader("X-Internal-Token"))
	if expected == "" || len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
