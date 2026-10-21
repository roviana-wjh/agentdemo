package flow

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type unifiedClassifierTestModel struct {
	content string
	err     error
}

func (m unifiedClassifierTestModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return schema.AssistantMessage(m.content, nil), nil
}

func (unifiedClassifierTestModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func TestUnifiedAssistantClassifyNormalizesModelOutput(t *testing.T) {
	assistant := &UnifiedAssistant{model: unifiedClassifierTestModel{content: "```json\n{\"intent\":\" BLOG_TOOL \",\"reason\":\"查询热门文章\"}\n```"}}
	result, err := assistant.Classify(context.Background(), "热门文章有哪些？")
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if result.Intent != IntentBlogTool || result.Reason != "查询热门文章" {
		t.Fatalf("result = %+v", result)
	}
}

func TestUnifiedAssistantClassifyFallsBackOnInvalidModelOutput(t *testing.T) {
	assistant := &UnifiedAssistant{model: unifiedClassifierTestModel{content: `{"intent":"unknown"}`}}
	result, err := assistant.Classify(context.Background(), "请看一下最近的评论")
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if result.Intent != IntentBlogTool || result.Reason != "fallback classification" {
		t.Fatalf("result = %+v", result)
	}

	result, err = assistant.Classify(context.Background(), "解释一下文章里的 RAG 原理")
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if result.Intent != IntentKnowledgeRAG {
		t.Fatalf("fallback intent = %q, want %q", result.Intent, IntentKnowledgeRAG)
	}
}

func TestUnifiedAssistantClassifyRejectsEmptyQueryAndModel(t *testing.T) {
	if _, err := (&UnifiedAssistant{}).Classify(context.Background(), " "); err == nil {
		t.Fatal("Classify() accepted an empty query")
	}
	if _, err := (&UnifiedAssistant{}).Classify(context.Background(), "hello"); err == nil {
		t.Fatal("Classify() accepted a nil model")
	}
}

func TestKeywordBasedIntentRoutesOnlyExplicitBlogDataQueries(t *testing.T) {
	for _, query := range []string{"最近的热门文章", "博客有多少篇文章", "最新评论在讨论什么"} {
		if got := keywordBasedIntent(query); got != IntentBlogTool {
			t.Errorf("keywordBasedIntent(%q) = %q, want %q", query, got, IntentBlogTool)
		}
	}
	for _, query := range []string{"如何实现 RAG", "解释一下缓存击穿", "为什么需要消息队列"} {
		if got := keywordBasedIntent(query); got != IntentKnowledgeRAG {
			t.Errorf("keywordBasedIntent(%q) = %q, want %q", query, got, IntentKnowledgeRAG)
		}
	}
}
