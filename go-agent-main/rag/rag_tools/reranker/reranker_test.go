package reranker

import (
	"context"
	"reflect"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestLexicalRerankerUsesQueryAndLimitsResults(t *testing.T) {
	ginDoc := &schema.Document{ID: "gin", Content: "Gin middleware example", MetaData: map[string]any{"title": "Gin middleware"}}
	otherDoc := &schema.Document{ID: "other", Content: "Cooking notes", MetaData: map[string]any{"title": "Dinner"}}
	ginDoc.WithScore(0.2)
	otherDoc.WithScore(0.3)

	results, err := NewLexicalReranker().Rerank(context.Background(), "Gin middleware", []*schema.Document{otherDoc, ginDoc}, 1)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "gin" {
		t.Fatalf("results = %+v", results)
	}
	if results[0].MetaData["rerank_mode"] != "lexical" {
		t.Fatalf("rerank_mode = %#v", results[0].MetaData["rerank_mode"])
	}
}

func TestParseOrderedIDsAcceptsMarkdownJSONFence(t *testing.T) {
	ids, err := parseOrderedIDs("```json\n{\"ordered_ids\":[\"b\",\"a\"]}\n```")
	if err != nil {
		t.Fatalf("parseOrderedIDs() error = %v", err)
	}
	if !reflect.DeepEqual(ids, []string{"b", "a"}) {
		t.Fatalf("ids = %#v", ids)
	}
}

func TestLLMRerankerFallsBackWhenModelIsNil(t *testing.T) {
	doc := &schema.Document{ID: "one", Content: "Go", MetaData: map[string]any{}}
	doc.WithScore(1)
	reranker := NewLLMReranker(nil, NewLexicalReranker(), 100)
	results, err := reranker.Rerank(context.Background(), "Go", []*schema.Document{doc}, 1)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "one" {
		t.Fatalf("results = %+v", results)
	}
}
