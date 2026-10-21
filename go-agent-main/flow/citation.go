package flow

import (
	"strconv"

	"github.com/cloudwego/eino/schema"
)

type Citation struct {
	Index          int     `json:"index"`
	ChunkID        string  `json:"chunk_id"`
	ArticleID      int64   `json:"article_id,omitempty"`
	Title          string  `json:"title,omitempty"`
	URL            string  `json:"url,omitempty"`
	ContentExcerpt string  `json:"content_excerpt,omitempty"`
	Score          float64 `json:"score"`
}

func BuildCitations(docs []*schema.Document) []Citation {
	results := make([]Citation, 0, len(docs))
	for index, doc := range docs {
		if doc == nil {
			continue
		}
		articleID := metadataInt64(doc.MetaData["article_id"])
		if doc.ID == "" {
			continue
		}
		title, _ := doc.MetaData["title"].(string)
		url, _ := doc.MetaData["url"].(string)
		results = append(results, Citation{
			Index:          index + 1,
			ChunkID:        doc.ID,
			ArticleID:      articleID,
			Title:          title,
			URL:            url,
			ContentExcerpt: truncateCitation(doc.Content, 180),
			Score:          doc.Score(),
		})
	}
	return results
}

func metadataInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case uint:
		return int64(typed)
	case uint64:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	default:
		return 0
	}
}

func truncateCitation(content string, maxRunes int) string {
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes]) + "…"
}
