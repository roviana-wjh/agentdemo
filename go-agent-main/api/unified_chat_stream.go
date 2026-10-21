package api

import (
	"go-agent/flow"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/gin-gonic/gin"
)

// UnifiedChatStream 统一的流式对话接口（意图识别 + RAG/Tool自动路由）
func UnifiedChatStream(c *gin.Context) {
	var req struct {
		Question  string `json:"question"`
		Query     string `json:"query"`
		SessionID string `json:"session_id"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 兼容question和query字段
	query := strings.TrimSpace(req.Query)
	if query == "" {
		query = strings.TrimSpace(req.Question)
	}
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question or query is required"})
		return
	}

	// 默认SessionID
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		req.SessionID = "default_user"
	}

	// 创建SSE Writer
	writer, err := NewSSEWriter(c, req.RequestID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}
	writer.Start()

	// 获取统一对话流
	unifiedFlow, err := flow.GetUnifiedChatFlow()
	if err != nil {
		writer.Error("unified_flow_not_ready", err.Error())
		return
	}

	// 设置追踪上下文
	ctx := langsmith.SetTrace(c.Request.Context(),
		langsmith.WithSessionName("GoAgent-Unified"),
		langsmith.AddTag("session:"+req.SessionID),
	)

	// 创建Citation通道
	citationSink := make(chan []flow.Citation, 1)
	routeSink := make(chan flow.IntentResultWithConfidence, 1)

	// 调用统一流（流式输出）
	stream, err := unifiedFlow.Stream(ctx, flow.UnifiedChatFlowInput{
		Query:        query,
		SessionID:    req.SessionID,
		CitationSink: citationSink,
		RouteSink:    routeSink,
	})
	if err != nil {
		writer.Error("unified_flow_stream_failed", err.Error())
		return
	}
	defer stream.Close()

	// 流式读取和发送
	var citations []flow.Citation
	var route *flow.IntentResultWithConfidence
	sendRoute := func() {
		if route != nil {
			return
		}
		select {
		case classified := <-routeSink:
			route = &classified
			writer.Event(SSEEventRoute, classified)
		default:
		}
	}
	for {
		sendRoute()
		select {
		case <-c.Request.Context().Done():
			return
		case classified := <-routeSink:
			if route == nil {
				route = &classified
				writer.Event(SSEEventRoute, classified)
			}
		case citations = <-citationSink:
			// 接收到citations，稍后发送
		default:
		}

		msg, recvErr := stream.Recv()
		sendRoute()

		// 再次尝试接收citations
		select {
		case citations = <-citationSink:
		default:
		}

		if recvErr != nil {
			if recvErr == io.EOF {
				sendRoute()
				// 流结束，发送Citations（如果有）
				for _, citation := range citations {
					writer.Citation(citation)
				}
				writer.Done()
				return
			}
			writer.Error("unified_flow_stream_read_failed", recvErr.Error())
			return
		}

		if msg != nil {
			for _, toolCall := range msg.ToolCalls {
				writer.Tool(map[string]any{
					"name":  toolCall.Function.Name,
					"input": toolCall.Function.Arguments,
					"id":    toolCall.ID,
				})
			}
			if msg.Content != "" {
				writer.Delta(msg.Content)
			}
		}
	}
}
