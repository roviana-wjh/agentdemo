package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatStreamRequest struct {
	Question  string        `json:"question" binding:"required"`
	History   []ChatMessage `json:"history,omitempty"`
	SessionID string        `json:"session_id,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
}

type AgentClient struct {
	baseURL               string
	chatStreamPath        string
	unifiedChatStreamPath string
	articleIndexPath      string
	articleIndexTimeout   time.Duration
	internalToken         string
	httpClient            *http.Client
}

func (c *AgentClient) WithInternalToken(token string) *AgentClient {
	clone := *c
	clone.internalToken = strings.TrimSpace(token)
	return &clone
}

func NewAgentClient(baseURL, chatStreamPath string, responseHeaderTimeout time.Duration) *AgentClient {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ResponseHeaderTimeout: responseHeaderTimeout,
		IdleConnTimeout:       90 * time.Second,
	}
	return NewAgentClientWithHTTPClient(baseURL, chatStreamPath, &http.Client{Transport: transport})
}

func NewAgentClientWithHTTPClient(baseURL, chatStreamPath string, httpClient *http.Client) *AgentClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &AgentClient{
		baseURL:               strings.TrimRight(baseURL, "/"),
		chatStreamPath:        "/" + strings.TrimLeft(chatStreamPath, "/"),
		unifiedChatStreamPath: "/v1/blog/assistant/stream",
		articleIndexPath:      "/internal/v1/index/articles",
		articleIndexTimeout:   90 * time.Second,
		httpClient:            httpClient,
	}
}

func (c *AgentClient) WithUnifiedChatStreamPath(path string) *AgentClient {
	clone := *c
	if strings.TrimSpace(path) != "" {
		clone.unifiedChatStreamPath = "/" + strings.TrimLeft(path, "/")
	}
	return &clone
}

func (c *AgentClient) WithArticleIndex(path string, timeout time.Duration) *AgentClient {
	clone := *c
	clone.articleIndexPath = "/" + strings.TrimLeft(path, "/")
	if timeout > 0 {
		clone.articleIndexTimeout = timeout
	}
	return &clone
}

func (c *AgentClient) ChatStream(ctx context.Context, input ChatStreamRequest) (*http.Response, error) {
	// The generic chat endpoint accepts "question", while the RAG endpoint
	// accepts "query". Send both so the configured upstream path can target
	// either endpoint without changing the gateway's public request contract.
	question := input.Question
	upstreamInput := struct {
		Question  string        `json:"question"`
		Query     string        `json:"query"`
		History   []ChatMessage `json:"history,omitempty"`
		SessionID string        `json:"session_id,omitempty"`
		RequestID string        `json:"request_id,omitempty"`
	}{
		Question:  question,
		Query:     question,
		History:   input.History,
		SessionID: input.SessionID,
		RequestID: input.RequestID,
	}
	body, err := json.Marshal(upstreamInput)
	if err != nil {
		return nil, fmt.Errorf("marshal agent request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.chatStreamPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create agent request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call agent service: %w", err)
	}
	return resp, nil
}

// UnifiedAssistantStream calls the intent-routing assistant endpoint. The
// response uses the same SSE envelope as the legacy RAG and blog endpoints.
func (c *AgentClient) UnifiedAssistantStream(ctx context.Context, input ChatStreamRequest) (*http.Response, error) {
	question := input.Question
	upstreamInput := struct {
		Question  string        `json:"question"`
		Query     string        `json:"query"`
		History   []ChatMessage `json:"history,omitempty"`
		SessionID string        `json:"session_id,omitempty"`
		RequestID string        `json:"request_id,omitempty"`
	}{
		Question: question, Query: question, History: input.History,
		SessionID: input.SessionID, RequestID: input.RequestID,
	}
	body, err := json.Marshal(upstreamInput)
	if err != nil {
		return nil, fmt.Errorf("marshal unified assistant request: %w", err)
	}
	path := c.unifiedChatStreamPath
	if path == "" {
		path = "/v1/blog/assistant/stream"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create unified assistant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call unified assistant service: %w", err)
	}
	return resp, nil
}

type WritingStreamRequest struct {
	Topic        string `json:"topic"`
	Requirements string `json:"requirements,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
}

func (c *AgentClient) WritingStream(ctx context.Context, input WritingStreamRequest) (*http.Response, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal writing request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/blog/writing/stream", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create writing request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-Internal-Token", c.internalToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call writing service: %w", err)
	}
	return resp, nil
}

func (c *AgentClient) InsightStream(ctx context.Context, path string, input any) (*http.Response, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal insight request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+strings.TrimLeft(path, "/"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create insight request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-Internal-Token", c.internalToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call insight service: %w", err)
	}
	return resp, nil
}

type SaveWritingDraftRequest struct {
	AuthorID  uint   `json:"author_id"`
	ArticleID uint   `json:"article_id,omitempty"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Image     string `json:"image,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

type DraftResult struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Version uint   `json:"version"`
	Status  string `json:"status"`
}

type ReviewIssue struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Keyword  string `json:"keyword,omitempty"`
	Position int    `json:"position,omitempty"`
}

type RiskCategory struct {
	Name        string `json:"name"`
	RiskLevel   string `json:"risk_level"`
	Description string `json:"description"`
}

type ArticleReviewRequest struct {
	ArticleID   uint     `json:"article_id"`
	Version     uint     `json:"version"`
	Source      string   `json:"source,omitempty"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	ContentHash string   `json:"content_hash"`
}

type ArticleReviewResult struct {
	ArticleID   uint           `json:"article_id"`
	Version     uint           `json:"version"`
	ContentHash string         `json:"content_hash"`
	Status      string         `json:"status"`
	RiskLevel   string         `json:"risk_level"`
	Score       int            `json:"score"`
	Categories  []RiskCategory `json:"categories"`
	Reasons     []string       `json:"reasons"`
	Suggestions []string       `json:"suggestions"`
	RuleHits    []ReviewIssue  `json:"rule_hits"`
}

func (c *AgentClient) ReviewArticle(ctx context.Context, input ArticleReviewRequest) (ArticleReviewResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return ArticleReviewResult{}, fmt.Errorf("marshal article review request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/blog/articles/review", bytes.NewReader(body))
	if err != nil {
		return ArticleReviewResult{}, fmt.Errorf("create article review request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ArticleReviewResult{}, fmt.Errorf("call article review service: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ArticleReviewResult{}, fmt.Errorf("read article review response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return ArticleReviewResult{}, fmt.Errorf("article review service returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	var envelope struct {
		Data ArticleReviewResult `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return ArticleReviewResult{}, fmt.Errorf("decode article review response: %w", err)
	}
	if envelope.Data.ArticleID != input.ArticleID || envelope.Data.ContentHash != input.ContentHash {
		return ArticleReviewResult{}, fmt.Errorf("article review response does not match the requested snapshot")
	}
	return envelope.Data, nil
}

func (c *AgentClient) SaveWritingDraft(ctx context.Context, input SaveWritingDraftRequest) (DraftResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return DraftResult{}, fmt.Errorf("marshal draft request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/blog/drafts/save", bytes.NewReader(body))
	if err != nil {
		return DraftResult{}, fmt.Errorf("create draft request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DraftResult{}, fmt.Errorf("call draft service: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return DraftResult{}, fmt.Errorf("read draft response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return DraftResult{}, fmt.Errorf("draft service returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	var envelope struct {
		Data DraftResult `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return DraftResult{}, fmt.Errorf("decode draft response: %w", err)
	}
	if envelope.Data.Status != "draft" {
		return DraftResult{}, fmt.Errorf("draft service returned unsafe status %q", envelope.Data.Status)
	}
	return envelope.Data, nil
}

type ArticleIndexRequest struct {
	ArticleID   int64      `json:"article_id"`
	Version     int        `json:"version"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Tags        []string   `json:"tags,omitempty"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	URL         string     `json:"url,omitempty"`
}

type ArticleIndexResponse struct {
	Success    bool            `json:"success"`
	ChunkCount int             `json:"chunk_count"`
	Result     json.RawMessage `json:"result,omitempty"`
}

func (c *AgentClient) IndexArticle(ctx context.Context, input ArticleIndexRequest) (ArticleIndexResponse, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return ArticleIndexResponse{}, fmt.Errorf("marshal article index request: %w", err)
	}
	if c.articleIndexTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.articleIndexTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.articleIndexPath, bytes.NewReader(body))
	if err != nil {
		return ArticleIndexResponse{}, fmt.Errorf("create article index request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ArticleIndexResponse{}, fmt.Errorf("call article index service: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ArticleIndexResponse{}, fmt.Errorf("read article index response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return ArticleIndexResponse{}, fmt.Errorf("article index service returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	var result ArticleIndexResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return ArticleIndexResponse{}, fmt.Errorf("decode article index response: %w", err)
	}
	if !result.Success {
		return result, fmt.Errorf("article index service reported failure")
	}
	return result, nil
}
