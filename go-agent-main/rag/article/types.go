package article

import (
	"fmt"
	"strings"
	"time"
)

type Snapshot struct {
	ArticleID   int64     `json:"article_id" binding:"required"`
	Version     int       `json:"version" binding:"required"`
	Title       string    `json:"title" binding:"required"`
	Content     string    `json:"content" binding:"required"`
	Tags        []string  `json:"tags,omitempty"`
	Status      string    `json:"status" binding:"required"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	URL         string    `json:"url,omitempty"`
}

func (s Snapshot) Validate() error {
	if s.ArticleID <= 0 {
		return fmt.Errorf("article_id must be positive")
	}
	if s.Version <= 0 {
		return fmt.Errorf("version must be positive")
	}
	if strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(s.Content) == "" {
		return fmt.Errorf("content is required")
	}
	if s.Status != "published" {
		return fmt.Errorf("only published articles can be indexed")
	}
	return nil
}

type BackendResult struct {
	IDs   []string `json:"ids,omitempty"`
	Error string   `json:"error,omitempty"`
}

type IndexResult struct {
	ChunkIDs []string      `json:"chunk_ids"`
	ES       BackendResult `json:"elasticsearch"`
	Milvus   BackendResult `json:"milvus"`
}

type IndexError struct {
	Result IndexResult
}

func (e *IndexError) Error() string {
	return "one or more article index backends failed"
}
