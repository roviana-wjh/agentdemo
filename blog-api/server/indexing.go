package server

import (
	agentclient "awesomeProject1/client"
	"awesomeProject1/model"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const (
	IndexStatusNotIndexed = "not_indexed"
	IndexStatusPending    = "pending"
	IndexStatusSuccess    = "success"
	IndexStatusFailed     = "failed"
)

type articleIndexClient interface {
	IndexArticle(ctx context.Context, input agentclient.ArticleIndexRequest) (agentclient.ArticleIndexResponse, error)
}

func hashArticleContent(title, content, image string) string {
	return hashArticleContentWithTags(title, content, image, nil)
}

func hashArticleContentWithTags(title, content, image string, tags model.Tags) string {
	tagsJSON, _ := json.Marshal(tags)
	hash := sha256.Sum256([]byte(title + "\x00" + content + "\x00" + image + "\x00" + string(tagsJSON)))
	return hex.EncodeToString(hash[:])
}

func buildArticleIndexRequest(article model.Article) agentclient.ArticleIndexRequest {
	return agentclient.ArticleIndexRequest{
		ArticleID:   int64(article.ID),
		Version:     int(article.Version),
		Title:       article.Title,
		Content:     article.Content,
		Tags:        append([]string(nil), article.Tags...),
		Status:      article.Status,
		PublishedAt: article.PublishedAt,
		URL:         fmt.Sprintf("/article/%d", article.ID),
	}
}

func syncArticleIndex(ctx context.Context, article model.Article, client articleIndexClient) error {
	_, err := client.IndexArticle(ctx, buildArticleIndexRequest(article))
	updates := map[string]any{"index_error": ""}
	if err != nil {
		updates["index_status"] = IndexStatusFailed
		updates["index_error"] = err.Error()
	} else {
		now := time.Now()
		updates["index_status"] = IndexStatusSuccess
		updates["indexed_at"] = &now
	}
	dbErr := model.DB.Model(&model.Article{}).
		Where("id = ? AND version = ?", article.ID, article.Version).
		Updates(updates).Error
	if dbErr != nil {
		return fmt.Errorf("update article index status: %w", dbErr)
	}
	return err
}
