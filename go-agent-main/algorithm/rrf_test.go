package algorithm

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestRRFFusionPromotesDocumentsFoundByBothRetrievers(t *testing.T) {
	sharedES := &schema.Document{ID: "shared", Content: "shared", MetaData: map[string]any{"source": "es"}}
	esOnly := &schema.Document{ID: "es-only", Content: "es", MetaData: map[string]any{}}
	sharedMilvus := &schema.Document{ID: "shared", Content: "shared", MetaData: map[string]any{"source": "milvus"}}
	milvusOnly := &schema.Document{ID: "milvus-only", Content: "milvus", MetaData: map[string]any{}}

	results, err := RRFFusionWithK(context.Background(), [][]*schema.Document{
		{esOnly, sharedES},
		{milvusOnly, sharedMilvus},
	}, 60)
	if err != nil {
		t.Fatalf("RRFFusionWithK() error = %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("result count = %d", len(results))
	}
	if results[0].ID != "shared" {
		t.Fatalf("first result = %s, want shared", results[0].ID)
	}
	if _, ok := results[0].MetaData["rrf_score"]; !ok {
		t.Fatalf("rrf_score metadata missing")
	}
}
