package flow

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestBuildCitationsPreservesRankAndChunkReferences(t *testing.T) {
	first := &schema.Document{
		ID:      "article:12:version:1:chunk:0",
		Content: strings.Repeat("中", 200),
		MetaData: map[string]any{
			"article_id": float64(12),
			"title":      "Gin Middleware",
			"url":        "/article/12",
		},
	}
	first.WithScore(0.9)
	second := &schema.Document{
		ID:       "article:12:version:1:chunk:1",
		Content:  "second chunk",
		MetaData: map[string]any{"article_id": "12", "title": "Gin Middleware"},
	}
	second.WithScore(0.8)

	citations := BuildCitations([]*schema.Document{first, second})
	if len(citations) != 2 {
		t.Fatalf("citation count = %d", len(citations))
	}
	if citations[0].Index != 1 || citations[0].ArticleID != 12 || citations[0].URL != "/article/12" {
		t.Fatalf("first citation = %+v", citations[0])
	}
	if citations[1].Index != 2 || citations[1].ChunkID != second.ID {
		t.Fatalf("second citation = %+v", citations[1])
	}
	if !strings.HasSuffix(citations[0].ContentExcerpt, "…") {
		t.Fatalf("excerpt was not truncated: %q", citations[0].ContentExcerpt)
	}
}

func TestMetadataInt64SupportsCommonJSONTypes(t *testing.T) {
	for _, value := range []any{12, int64(12), uint(12), uint64(12), float64(12), "12"} {
		if got := metadataInt64(value); got != 12 {
			t.Fatalf("metadataInt64(%T) = %d", value, got)
		}
	}
}

func TestBuildCitationsKeepsPromptIndexesWhenInvalidDocsAreSkipped(t *testing.T) {
	valid := &schema.Document{
		ID:       "article:12:version:1:chunk:2",
		Content:  "valid chunk",
		MetaData: map[string]any{"article_id": 12},
	}

	citations := BuildCitations([]*schema.Document{nil, {Content: "missing id"}, valid})
	if len(citations) != 1 {
		t.Fatalf("citation count = %d", len(citations))
	}
	if citations[0].Index != 3 {
		t.Fatalf("citation index = %d, want 3", citations[0].Index)
	}
}
