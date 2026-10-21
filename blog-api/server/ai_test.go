package server

import (
	agentclient "awesomeProject1/client"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAIChatStreamProxiesAgentSSE(t *testing.T) {
	var upstreamRequest agentclient.ChatStreamRequest
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/stream" {
			t.Errorf("agent path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&upstreamRequest); err != nil {
			t.Errorf("decode upstream request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("X-Request-ID", "agent-request-1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event:start\ndata:{\"type\":\"start\"}\n\n"))
		_, _ = w.Write([]byte("event:delta\ndata:{\"type\":\"delta\",\"content\":\"hello\"}\n\n"))
		_, _ = w.Write([]byte("event:done\ndata:{\"type\":\"done\"}\n\n"))
	}))
	defer agent.Close()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/chat/stream", strings.NewReader(`{"question":"hello"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("userId", uint(7))

	client := agentclient.NewAgentClientWithHTTPClient(agent.URL, "/v1/chat/stream", agent.Client())
	newAIChatStreamHandler(client)(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := recorder.Header().Get("X-Request-ID"); got != "agent-request-1" {
		t.Fatalf("X-Request-ID = %q", got)
	}
	if upstreamRequest.SessionID != "user-7" {
		t.Fatalf("session_id = %q, want user-7", upstreamRequest.SessionID)
	}
	for _, expected := range []string{"event:start", "event:delta", "hello", "event:done"} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("proxy response does not contain %q: %s", expected, recorder.Body.String())
		}
	}
}

func TestUnifiedAIChatStreamProxiesAgentSSE(t *testing.T) {
	var upstreamRequest agentclient.ChatStreamRequest
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/blog/assistant/stream" {
			t.Errorf("agent path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&upstreamRequest); err != nil {
			t.Errorf("decode upstream request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("X-Request-ID", "unified-request-1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event:start\ndata:{\"type\":\"start\"}\n\n"))
		_, _ = w.Write([]byte("event:route\ndata:{\"type\":\"route\",\"data\":{\"intent\":\"blog_tool\"}}\n\n"))
		_, _ = w.Write([]byte("event:delta\ndata:{\"type\":\"delta\",\"content\":\"hello\"}\n\n"))
		_, _ = w.Write([]byte("event:done\ndata:{\"type\":\"done\"}\n\n"))
	}))
	defer agent.Close()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/assistant/stream", strings.NewReader(`{"question":"最近的热门文章？","session_id":"chat-1"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("userId", uint(9))

	client := agentclient.NewAgentClientWithHTTPClient(agent.URL, "/v1/rag/chat/stream", agent.Client())
	newAIStreamProxyHandler(client, client.UnifiedAssistantStream)(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("X-Request-ID"); got != "unified-request-1" {
		t.Fatalf("X-Request-ID = %q", got)
	}
	if upstreamRequest.SessionID != "user-9:chat-1" {
		t.Fatalf("session_id = %q, want user-9:chat-1", upstreamRequest.SessionID)
	}
	for _, expected := range []string{"event:start", "event:route", "blog_tool", "event:delta", "hello", "event:done"} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("proxy response does not contain %q: %s", expected, recorder.Body.String())
		}
	}
}

func TestAIChatStreamRejectsMissingUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/chat/stream", strings.NewReader(`{"question":"hello"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	newAIChatStreamHandler(agentclient.NewAgentClientWithHTTPClient("http://127.0.0.1", "/v1/chat/stream", nil))(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestNamespaceSessionIDPreventsCrossUserCollisions(t *testing.T) {
	first := namespaceSessionID(uint(7), "shared-session")
	second := namespaceSessionID(uint(8), "shared-session")
	if first != "user-7:shared-session" {
		t.Fatalf("first session = %q", first)
	}
	if second != "user-8:shared-session" {
		t.Fatalf("second session = %q", second)
	}
	if first == second {
		t.Fatal("different users share the same namespaced session")
	}
}
