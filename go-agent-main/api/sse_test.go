package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSSEWriterUsesJSONEventContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writer, err := NewSSEWriter(ctx, "req-123", "session-456")
	if err != nil {
		t.Fatalf("NewSSEWriter() error = %v", err)
	}
	writer.Start()
	writer.Delta("hello")
	writer.Done()

	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"event:start",
		`"type":"start"`,
		`"request_id":"req-123"`,
		`"session_id":"session-456"`,
		"event:delta",
		`"content":"hello"`,
		"event:done",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("response body does not contain %q: %s", expected, body)
		}
	}
	if got := recorder.Header().Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("X-Request-ID = %q, want req-123", got)
	}
}

func TestChatStreamV1RejectsMissingQuestion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/stream", strings.NewReader(`{"session_id":"session-1"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	ChatStreamV1(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("invalid request unexpectedly started SSE stream")
	}
}

func TestSSEWriterSupportsStructuredEventsAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writer, err := NewSSEWriter(ctx, "req-structured", "")
	if err != nil {
		t.Fatalf("NewSSEWriter() error = %v", err)
	}
	writer.Start()
	writer.Citation(map[string]any{"article_id": 12})
	writer.Tool(map[string]any{"name": "search_article"})
	writer.Error("model_failed", "model unavailable")

	body := recorder.Body.String()
	for _, expected := range []string{
		"event:citation",
		`"article_id":12`,
		"event:tool",
		`"name":"search_article"`,
		"event:error",
		`"code":"model_failed"`,
		`"message":"model unavailable"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("response body does not contain %q: %s", expected, body)
		}
	}
}

func TestBuildChatMessagesIgnoresUnknownRoles(t *testing.T) {
	messages := buildChatMessages(ChatTestRequest{
		Question: "current",
		History: []ChatTestMessage{
			{Role: "user", Content: "first"},
			{Role: "tool", Content: "ignored"},
			{Role: "assistant", Content: "second"},
		},
	})

	if len(messages) != 4 {
		t.Fatalf("message count = %d, want 4", len(messages))
	}
	if got := messages[len(messages)-1].Content; got != "current" {
		t.Fatalf("last message = %q, want current", got)
	}
}
