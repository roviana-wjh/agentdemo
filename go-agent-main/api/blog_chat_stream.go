package api

import (
	"go-agent/flow"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

func BlogChatStream(c *gin.Context) {
	var req struct {
		Question  string `json:"question"`
		Query     string `json:"query"`
		SessionID string `json:"session_id,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	question := strings.TrimSpace(req.Query)
	if question == "" {
		question = strings.TrimSpace(req.Question)
	}
	if question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question or query is required"})
		return
	}

	writer, err := NewSSEWriter(c, req.RequestID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}
	writer.Start()

	assistant, err := flow.GetBlogAssistant()
	if err != nil {
		writer.Error("blog_assistant_not_ready", err.Error())
		return
	}
	stream, err := assistant.Stream(c.Request.Context(), []*schema.Message{schema.UserMessage(question)})
	if err != nil {
		writer.Error("blog_assistant_stream_failed", err.Error())
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
			writer.Error("blog_assistant_stream_read_failed", recvErr.Error())
			return
		}
		if message != nil && message.Content != "" {
			writer.Delta(message.Content)
		}
	}
}
