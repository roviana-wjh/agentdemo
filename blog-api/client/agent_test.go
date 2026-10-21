package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAgentClientChatStream(t *testing.T) {
	var received ChatStreamRequest
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/chat/stream" {
			t.Errorf("path = %s, want /v1/chat/stream", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("Accept = %q", got)
		}
		var payload struct {
			ChatStreamRequest
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		received = payload.ChatStreamRequest
		receivedQuery = payload.Query
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event:done\ndata:{\"type\":\"done\"}\n\n"))
	}))
	defer server.Close()

	client := NewAgentClientWithHTTPClient(server.URL, "v1/chat/stream", server.Client())
	resp, err := client.ChatStream(context.Background(), ChatStreamRequest{
		Question:  "hello",
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if received.Question != "hello" || received.SessionID != "session-1" {
		t.Fatalf("received request = %+v", received)
	}
	if receivedQuery != "hello" {
		t.Fatalf("query = %q, want hello", receivedQuery)
	}
}

func TestAgentClientUnifiedAssistantStreamUsesConfiguredPath(t *testing.T) {
	var received ChatStreamRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/blog/assistant/stream" {
			t.Errorf("path = %q, want /v1/blog/assistant/stream", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event:route\ndata:{\"type\":\"route\"}\n\n"))
	}))
	defer server.Close()

	client := NewAgentClientWithHTTPClient(server.URL, "/v1/rag/chat/stream", server.Client())
	resp, err := client.UnifiedAssistantStream(context.Background(), ChatStreamRequest{
		Question:  "最近有哪些热门文章？",
		SessionID: "user-7:session-1",
	})
	if err != nil {
		t.Fatalf("UnifiedAssistantStream() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if received.Question != "最近有哪些热门文章？" || received.SessionID != "user-7:session-1" {
		t.Fatalf("received request = %+v", received)
	}
}

func TestAgentClientUnifiedAssistantStreamSupportsCustomPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/assistant" {
			t.Errorf("path = %q, want /custom/assistant", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event:done\ndata:{}\n\n"))
	}))
	defer server.Close()

	client := NewAgentClientWithHTTPClient(server.URL, "/v1/rag/chat/stream", server.Client()).
		WithUnifiedChatStreamPath("custom/assistant")
	resp, err := client.UnifiedAssistantStream(context.Background(), ChatStreamRequest{Question: "hello"})
	if err != nil {
		t.Fatalf("UnifiedAssistantStream() error = %v", err)
	}
	defer resp.Body.Close()
}

func TestAgentClientIndexArticle(t *testing.T) {
	var received ArticleIndexRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/index/articles" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"chunk_count":2,"result":{"chunk_ids":["a","b"]}}`))
	}))
	defer server.Close()

	client := NewAgentClientWithHTTPClient(server.URL, "/v1/chat/stream", server.Client()).
		WithArticleIndex("/internal/v1/index/articles", time.Second)
	result, err := client.IndexArticle(context.Background(), ArticleIndexRequest{
		ArticleID: 12,
		Version:   3,
		Title:     "Gin",
		Content:   "Article body",
		Status:    "published",
	})
	if err != nil {
		t.Fatalf("IndexArticle() error = %v", err)
	}
	if result.ChunkCount != 2 || received.ArticleID != 12 || received.Version != 3 {
		t.Fatalf("result = %+v, received = %+v", result, received)
	}
}

func TestAgentClientReviewArticle(t *testing.T) {
	var received ArticleReviewRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/blog/articles/review" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("X-Internal-Token") != "secret" {
			t.Errorf("missing internal token")
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"article_id":7,"version":2,"content_hash":"hash-7","status":"approved","risk_level":"low","score":96,"categories":[],"reasons":[],"suggestions":[],"rule_hits":[]}}`))
	}))
	defer server.Close()

	client := NewAgentClientWithHTTPClient(server.URL, "/v1/chat/stream", server.Client()).WithInternalToken("secret")
	result, err := client.ReviewArticle(context.Background(), ArticleReviewRequest{
		ArticleID: 7, Version: 2, Title: "Go", Content: "body", ContentHash: "hash-7",
	})
	if err != nil {
		t.Fatalf("ReviewArticle() error = %v", err)
	}
	if result.Status != "approved" || result.Score != 96 || received.ArticleID != 7 {
		t.Fatalf("result=%+v received=%+v", result, received)
	}
}

func TestAgentClientReviewArticleRejectsMismatchedSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"article_id":7,"content_hash":"stale","status":"approved"}}`))
	}))
	defer server.Close()

	client := NewAgentClientWithHTTPClient(server.URL, "/v1/chat/stream", server.Client())
	_, err := client.ReviewArticle(context.Background(), ArticleReviewRequest{ArticleID: 7, ContentHash: "current"})
	if err == nil {
		t.Fatal("ReviewArticle() accepted a mismatched snapshot")
	}
}
