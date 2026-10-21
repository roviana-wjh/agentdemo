package server

import (
	agentclient "awesomeProject1/client"
	"awesomeProject1/setting"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var defaultAgentClient = agentclient.NewAgentClient(
	setting.AgentSetting.BaseURL,
	setting.AgentSetting.ChatStreamPath,
	setting.AgentSetting.ResponseHeaderTimeout,
).WithArticleIndex(
	setting.AgentSetting.ArticleIndexPath,
	setting.AgentSetting.ArticleIndexTimeout,
).WithUnifiedChatStreamPath(
	setting.AgentSetting.UnifiedChatStreamPath,
).WithInternalToken(setting.InternalToolSetting.Token)

func AIChatStream(c *gin.Context) {
	newAIChatStreamHandler(defaultAgentClient)(c)
}

// UnifiedAIChatStream is the new public gateway entry point. The legacy
// /api/ai/chat/stream handler remains unchanged for existing clients.
func UnifiedAIChatStream(c *gin.Context) {
	newAIStreamProxyHandler(defaultAgentClient, defaultAgentClient.UnifiedAssistantStream)(c)
}

func newAIStreamProxyHandler(client *agentclient.AgentClient, call func(context.Context, agentclient.ChatStreamRequest) (*http.Response, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil || call == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI agent client is not configured"})
			return
		}
		var req agentclient.ChatStreamRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if strings.TrimSpace(req.Question) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
			return
		}
		req.Question = strings.TrimSpace(req.Question)
		userID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		req.SessionID = namespaceSessionID(userID, req.SessionID)
		if req.RequestID == "" {
			req.RequestID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}
		resp, err := call(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		if resp == nil || resp.Body == nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "agent service returned an empty response"})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			message := strings.TrimSpace(string(body))
			if message == "" {
				message = resp.Status
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": "agent service rejected request", "agent_status": resp.StatusCode, "detail": message})
			return
		}
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			c.JSON(http.StatusBadGateway, gin.H{"error": "agent service returned a non-SSE response"})
			return
		}
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		if requestID := resp.Header.Get("X-Request-ID"); requestID != "" {
			c.Header("X-Request-ID", requestID)
		}
		c.Status(http.StatusOK)
		c.Writer.Flush()
		buffer := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buffer)
			if n > 0 {
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					log.Printf("AI unified SSE write failed: %v", writeErr)
					return
				}
				c.Writer.Flush()
			}
			if readErr != nil {
				if readErr != io.EOF {
					log.Printf("AI unified SSE upstream read failed: %v", readErr)
				}
				return
			}
		}
	}
}

func newAIChatStreamHandler(client *agentclient.AgentClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req agentclient.ChatStreamRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if strings.TrimSpace(req.Question) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
			return
		}
		req.Question = strings.TrimSpace(req.Question)

		userID, exists := c.Get("userId")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		req.SessionID = namespaceSessionID(userID, req.SessionID)
		if req.RequestID == "" {
			req.RequestID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}

		resp, err := client.ChatStream(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			message := strings.TrimSpace(string(body))
			if message == "" {
				message = resp.Status
			}
			c.JSON(http.StatusBadGateway, gin.H{
				"error":        "agent service rejected request",
				"agent_status": resp.StatusCode,
				"detail":       message,
			})
			return
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			c.JSON(http.StatusBadGateway, gin.H{"error": "agent service returned a non-SSE response"})
			return
		}

		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		if requestID := resp.Header.Get("X-Request-ID"); requestID != "" {
			c.Header("X-Request-ID", requestID)
		}
		c.Status(http.StatusOK)
		c.Writer.Flush()

		buffer := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buffer)
			if n > 0 {
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					log.Printf("AI chat SSE write failed: %v", writeErr)
					return
				}
				c.Writer.Flush()
			}
			if readErr != nil {
				if readErr != io.EOF {
					log.Printf("AI chat SSE upstream read failed: %v", readErr)
				}
				return
			}
		}
	}
}

func namespaceSessionID(userID any, clientSessionID string) string {
	clientSessionID = strings.TrimSpace(clientSessionID)
	if clientSessionID == "" {
		return fmt.Sprintf("user-%v", userID)
	}
	// Always prepend the authenticated user boundary. Never trust a client
	// supplied session ID as a globally unique Redis key.
	return fmt.Sprintf("user-%v:%s", userID, clientSessionID)
}
