package api

import (
	"go-agent/config"
	"go-agent/model/chat_model"
	"io"
	"net/http"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

// ChatTestRequest 聊天测试请求结构
type ChatTestRequest struct {
	Question  string            `json:"question" binding:"required"`
	History   []ChatTestMessage `json:"history,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// ChatTestMessage 聊天消息结构（用于前端传递）
type ChatTestMessage struct {
	Role    string `json:"role"` // "user" 或 "assistant"
	Content string `json:"content"`
}

// ChatTestResponse 聊天测试响应结构
type ChatTestResponse struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// ChatGenerate 聊天模型的常规输出
func ChatGenerate(c *gin.Context) {
	var req ChatTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// 使用请求的上下文
	ctx := c.Request.Context()

	messages := buildChatMessages(req)

	// 调用模型的 Generate 方法
	chat, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to init chat model: " + err.Error()})
		return
	}
	response, err := chat.Generate(ctx, messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate answer: " + err.Error()})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, ChatTestResponse{
		Question: req.Question,
		Answer:   response.Content,
	})
}

// ChatStream 测试聊天模型的流式输出
func ChatStream(c *gin.Context) {
	streamChat(c)
}

// ChatStreamV1 exposes the stable streaming API used by the Blog gateway.
func ChatStreamV1(c *gin.Context) {
	streamChat(c)
}

func streamChat(c *gin.Context) {
	var req ChatTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	writer, err := NewSSEWriter(c, req.RequestID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	messages := buildChatMessages(req)

	chat, err := chat_model.GetChatModel(c.Request.Context(), config.Cfg.ChatModelType)
	if err != nil {
		writer.Start()
		writer.Error("model_init_failed", err.Error())
		return
	}
	streamReader, err := chat.Stream(c.Request.Context(), messages)
	if err != nil {
		writer.Start()
		writer.Error("model_stream_failed", err.Error())
		return
	}
	defer streamReader.Close()

	writer.Start()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		msg, err := streamReader.Recv()
		if err != nil {
			if err == io.EOF {
				writer.Done()
				return
			}
			writer.Error("stream_read_failed", err.Error())
			return
		}

		if msg != nil && msg.Content != "" {
			writer.Delta(msg.Content)
		}
	}
}

func buildChatMessages(req ChatTestRequest) []*schema.Message {
	messages := []*schema.Message{schema.SystemMessage("你是一个有用的AI助手。")}
	for _, msg := range req.History {
		switch msg.Role {
		case "user":
			messages = append(messages, schema.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(msg.Content, []schema.ToolCall{}))
		}
	}
	return append(messages, schema.UserMessage(req.Question))
}
