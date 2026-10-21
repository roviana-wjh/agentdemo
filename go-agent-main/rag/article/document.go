package article

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

func BuildDocuments(ctx context.Context, snapshot Snapshot, splitter document.Transformer) ([]*schema.Document, error) {
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	if splitter == nil {
		return nil, fmt.Errorf("document splitter is not initialized")
	}

	content := strings.TrimSpace(snapshot.Title + "\n\n" + CleanMarkdown(snapshot.Content))
	baseID := fmt.Sprintf("article:%d:version:%d", snapshot.ArticleID, snapshot.Version)
	metadata := map[string]any{
		"article_id": snapshot.ArticleID,
		"version":    snapshot.Version,
		"title":      snapshot.Title,
		"tags":       snapshot.Tags,
		"status":     snapshot.Status,
		"url":        snapshot.URL,
	}
	if !snapshot.PublishedAt.IsZero() {
		metadata["published_at"] = snapshot.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	docs, err := splitter.Transform(ctx, []*schema.Document{{
		ID:       baseID,
		Content:  content,
		MetaData: metadata,
	}})
	if err != nil {
		return nil, fmt.Errorf("split article: %w", err)
	}
	for index, doc := range docs {
		doc.ID = fmt.Sprintf("%s:chunk:%d", baseID, index)
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		doc.MetaData["chunk_id"] = doc.ID
		doc.MetaData["chunk_index"] = index
	}
	return docs, nil
}
