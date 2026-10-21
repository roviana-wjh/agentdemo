package blog_tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxResponseBody = 2 << 20

type Article struct {
	ID           uint       `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content,omitempty"`
	Image        string     `json:"image,omitempty"`
	AuthorID     uint       `json:"author_id,omitempty"`
	AuthorName   string     `json:"author_name,omitempty"`
	ViewCount    int        `json:"view_count,omitempty"`
	CommentCount int64      `json:"comment_count,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	Version      uint       `json:"version,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

type Comment struct {
	ID           uint      `json:"id"`
	Content      string    `json:"content"`
	UserID       uint      `json:"user_id"`
	UserName     string    `json:"user_name,omitempty"`
	ArticleID    uint      `json:"article_id"`
	ArticleTitle string    `json:"article_title,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type SearchArticleResult struct {
	Articles []Article `json:"articles"`
}

type GetArticleDetailResult struct {
	Article Article `json:"article"`
}

type GetHotArticlesResult struct {
	Articles []Article `json:"articles"`
}

type TagStat struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type GetHotTagsResult struct {
	Tags []TagStat `json:"tags"`
}

type BlogStatistics struct {
	ArticleCount int64 `json:"article_count"`
	TotalViews   int64 `json:"total_views"`
	CommentCount int64 `json:"comment_count"`
}

type GetRecentCommentsResult struct {
	Comments []Comment `json:"comments"`
}

type Client struct {
	baseURL       *url.URL
	internalToken string
	httpClient    *http.Client
}

func NewClient(baseURL, internalToken string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid blog API base URL %q", baseURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported blog API URL scheme %q", parsed.Scheme)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:       parsed,
		internalToken: strings.TrimSpace(internalToken),
		httpClient:    httpClient,
	}, nil
}

func (c *Client) SearchArticle(ctx context.Context, keyword string, limit int) (SearchArticleResult, error) {
	var envelope struct {
		Data []Article `json:"data"`
	}
	err := c.get(ctx, "/internal/v1/tools/search_article", url.Values{
		"q": {keyword}, "limit": {strconv.Itoa(limit)},
	}, &envelope)
	return SearchArticleResult{Articles: envelope.Data}, err
}

func (c *Client) GetArticleDetail(ctx context.Context, articleID uint) (GetArticleDetailResult, error) {
	var envelope struct {
		Data Article `json:"data"`
	}
	err := c.get(ctx, "/internal/v1/tools/get_article_detail", url.Values{
		"id": {strconv.FormatUint(uint64(articleID), 10)},
	}, &envelope)
	return GetArticleDetailResult{Article: envelope.Data}, err
}

func (c *Client) GetHotArticles(ctx context.Context, limit int) (GetHotArticlesResult, error) {
	var envelope struct {
		Data []Article `json:"data"`
	}
	err := c.get(ctx, "/internal/v1/tools/get_hot_articles", url.Values{
		"limit": {strconv.Itoa(limit)},
	}, &envelope)
	return GetHotArticlesResult{Articles: envelope.Data}, err
}

func (c *Client) GetHotTags(ctx context.Context, limit int) (GetHotTagsResult, error) {
	var envelope struct {
		Data []TagStat `json:"data"`
	}
	err := c.get(ctx, "/internal/v1/tools/get_hot_tags", url.Values{
		"limit": {strconv.Itoa(limit)},
	}, &envelope)
	return GetHotTagsResult{Tags: envelope.Data}, err
}

func (c *Client) GetBlogStatistics(ctx context.Context) (BlogStatistics, error) {
	var envelope struct {
		Data BlogStatistics `json:"data"`
	}
	err := c.get(ctx, "/internal/v1/tools/get_blog_statistics", nil, &envelope)
	return envelope.Data, err
}

func (c *Client) GetRecentComments(ctx context.Context, articleID uint, limit int) (GetRecentCommentsResult, error) {
	query := url.Values{"limit": {strconv.Itoa(limit)}}
	if articleID > 0 {
		query.Set("article_id", strconv.FormatUint(uint64(articleID), 10))
	}
	var envelope struct {
		Data []Comment `json:"data"`
	}
	err := c.get(ctx, "/internal/v1/tools/get_recent_comments", query, &envelope)
	return GetRecentCommentsResult{Comments: envelope.Data}, err
}

func (c *Client) get(ctx context.Context, path string, query url.Values, output any) error {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create blog API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call blog API: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody+1))
	if err != nil {
		return fmt.Errorf("read blog API response: %w", err)
	}
	if len(body) > maxResponseBody {
		return fmt.Errorf("blog API response exceeds %d bytes", maxResponseBody)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("blog API returned %s: %s", resp.Status, message)
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("decode blog API response: %w", err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, input, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode blog API request: %w", err)
	}
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create blog API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call blog API: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody+1))
	if err != nil {
		return fmt.Errorf("read blog API response: %w", err)
	}
	if len(responseBody) > maxResponseBody {
		return fmt.Errorf("blog API response exceeds %d bytes", maxResponseBody)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("blog API returned %s: %s", resp.Status, message)
	}
	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode blog API response: %w", err)
	}
	return nil
}
