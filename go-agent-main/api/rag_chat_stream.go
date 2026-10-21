package api

import (
	"go-agent/flow"
	"io"

	"github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/gin-gonic/gin"
)

func RAGChatStream(c *gin.Context) {
	var req struct {
		Query     string `json:"query" binding:"required"`
		SessionID string `json:"session_id"`
		RequestID string `json:"request_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if req.SessionID == "" {
		req.SessionID = "default_user"
	}
	writer, err := NewSSEWriter(c, req.RequestID, req.SessionID)
	if err != nil {
		c.JSON(500, gin.H{"error": "streaming not supported"})
		return
	}
	writer.Start()

	ctx := langsmith.SetTrace(c.Request.Context(),
		langsmith.WithSessionName("GoAgent"),
		langsmith.AddTag("session:"+req.SessionID),
	)

	ragRunner, err := flow.GetRAGChatFlow()
	if err != nil {
		writer.Error("rag_not_ready", err.Error())
		return
	}

	// 使用 Stream 模式调用图
	citationSink := make(chan []flow.Citation, 1)
	stream, err := ragRunner.Stream(ctx, flow.RAGChatInput{
		Query:        req.Query,
		SessionID:    req.SessionID,
		CitationSink: citationSink,
	})
	if err != nil {
		writer.Error("rag_stream_failed", err.Error())
		return
	}
	defer stream.Close()

	var citations []flow.Citation
	captureCitations := func() {
		select {
		case citations = <-citationSink:
		default:
		}
	}
	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		msg, err := stream.Recv()
		captureCitations()
		if err != nil {
			if err == io.EOF {
				captureCitations()
				for _, citation := range citations {
					writer.Citation(citation)
				}
				writer.Done()
				return
			}
			writer.Error("rag_stream_read_failed", err.Error())
			return
		}

		if msg != nil && msg.Content != "" {
			writer.Delta(msg.Content)
		}
	}
}
