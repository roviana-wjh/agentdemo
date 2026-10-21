package server

import (
	agentclient "awesomeProject1/client"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AIWritingStream(c *gin.Context) {
	var req agentclient.WritingStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Topic) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
		return
	}
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("writing-user-%v", userID)
	}
	resp, err := defaultAgentClient.WritingStream(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	proxySSE(c, resp)
}

func SaveAIWritingDraft(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		ArticleID uint   `json:"article_id,omitempty"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Image     string `json:"image,omitempty"`
		Confirmed bool   `json:"confirmed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !req.Confirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "explicit confirmation is required"})
		return
	}
	result, err := defaultAgentClient.SaveWritingDraft(c.Request.Context(), agentclient.SaveWritingDraftRequest{
		AuthorID: userID.(uint), ArticleID: req.ArticleID, Title: req.Title,
		Content: req.Content, Image: req.Image, Confirmed: true,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func proxySSE(c *gin.Context, resp *http.Response) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent service rejected request", "detail": strings.TrimSpace(string(body))})
		return
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent service returned a non-SSE response"})
		return
	}
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()
	buffer := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
				log.Printf("AI SSE proxy write failed: %v", writeErr)
				return
			}
			c.Writer.Flush()
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("AI SSE proxy upstream read failed: %v", readErr)
			}
			return
		}
	}
}
