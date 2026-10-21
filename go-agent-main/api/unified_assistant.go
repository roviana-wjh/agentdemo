package api

import (
	"go-agent/flow"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

func UnifiedAssistantStream(c *gin.Context) {
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
	query := strings.TrimSpace(req.Question)
	if query == "" {
		query = strings.TrimSpace(req.Query)
	}
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question or query is required"})
		return
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" {
		req.SessionID = "default_user"
	}
	writer, err := NewSSEWriter(c, req.RequestID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}
	writer.Start()
	assistant, err := flow.GetUnifiedAssistant()
	if err != nil {
		writer.Error("unified_assistant_not_ready", err.Error())
		return
	}
	intent, err := assistant.Classify(c.Request.Context(), query)
	if err != nil {
		writer.Error("intent_classification_failed", err.Error())
		return
	}
	writer.Event(SSEEventRoute, intent)

	switch intent.Intent {
	case flow.IntentBlogTool:
		blogAssistant, getErr := flow.GetBlogAssistant()
		if getErr != nil {
			writer.Error("blog_assistant_not_ready", getErr.Error())
			return
		}
		stream, streamErr := blogAssistant.Stream(c.Request.Context(), []*schema.Message{schema.UserMessage(query)})
		if streamErr != nil {
			writer.Error("blog_assistant_stream_failed", streamErr.Error())
			return
		}
		defer stream.Close()
		streamMessages(c, writer, stream, "blog_tool")
	default:
		ragRunner, getErr := flow.GetRAGChatFlow()
		if getErr != nil {
			writer.Error("rag_not_ready", getErr.Error())
			return
		}
		ctx := langsmith.SetTrace(c.Request.Context(), langsmith.WithSessionName("GoAgent"), langsmith.AddTag("session:"+req.SessionID))
		citationSink := make(chan []flow.Citation, 1)
		stream, streamErr := ragRunner.Stream(ctx, flow.RAGChatInput{Query: query, SessionID: req.SessionID, CitationSink: citationSink})
		if streamErr != nil {
			writer.Error("rag_stream_failed", streamErr.Error())
			return
		}
		defer stream.Close()
		var citations []flow.Citation
		for {
			msg, recvErr := stream.Recv()
			select {
			case citations = <-citationSink:
			default:
			}
			if recvErr != nil {
				if recvErr == io.EOF {
					select {
					case citations = <-citationSink:
					default:
					}
					for _, citation := range citations {
						writer.Citation(citation)
					}
					writer.Done()
					return
				}
				writer.Error("rag_stream_read_failed", recvErr.Error())
				return
			}
			if msg != nil && msg.Content != "" {
				writer.Delta(msg.Content)
			}
		}
	}
}

func streamMessages(c *gin.Context, writer *SSEWriter, stream *schema.StreamReader[*schema.Message], route string) {
	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
		message, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				writer.Done()
				return
			}
			writer.Error(route+"_stream_read_failed", err.Error())
			return
		}
		if message != nil {
			for _, toolCall := range message.ToolCalls {
				writer.Tool(map[string]any{
					"name":  toolCall.Function.Name,
					"input": toolCall.Function.Arguments,
					"id":    toolCall.ID,
				})
			}
			if message.Content != "" {
				writer.Delta(message.Content)
			}
		}
	}
}
