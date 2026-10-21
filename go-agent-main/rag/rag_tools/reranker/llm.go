package reranker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type LLMReranker struct {
	model            model.BaseChatModel
	fallback         Reranker
	maxContentLength int
}

func NewLLMReranker(chatModel model.BaseChatModel, fallback Reranker, maxContentLength int) *LLMReranker {
	if maxContentLength <= 0 {
		maxContentLength = 800
	}
	return &LLMReranker{model: chatModel, fallback: fallback, maxContentLength: maxContentLength}
}

func (r *LLMReranker) Rerank(ctx context.Context, query string, docs []*schema.Document, topK int) ([]*schema.Document, error) {
	fallbackResults, _ := r.fallback.Rerank(ctx, query, docs, len(docs))
	if r.model == nil || len(docs) == 0 {
		return limit(fallbackResults, topK), nil
	}

	type candidate struct {
		ID      string `json:"id"`
		Title   string `json:"title,omitempty"`
		Content string `json:"content"`
	}
	candidates := make([]candidate, 0, len(docs))
	for _, doc := range docs {
		title, _ := doc.MetaData["title"].(string)
		candidates = append(candidates, candidate{
			ID:      doc.ID,
			Title:   title,
			Content: truncateRunes(doc.Content, r.maxContentLength),
		})
	}
	candidateJSON, err := json.Marshal(candidates)
	if err != nil {
		return limit(fallbackResults, topK), nil
	}
	prompt := fmt.Sprintf(`根据用户问题对候选文章片段按相关性排序。
只输出 JSON，格式为 {"ordered_ids":["id1","id2"]}，不得解释。
用户问题：%s
候选片段：%s`, query, candidateJSON)
	response, err := r.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是博客检索结果重排器，只按问题相关性排序候选片段。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return limit(fallbackResults, topK), nil
	}

	orderedIDs, err := parseOrderedIDs(response.Content)
	if err != nil || len(orderedIDs) == 0 {
		return limit(fallbackResults, topK), nil
	}
	docByID := make(map[string]*schema.Document, len(fallbackResults))
	for _, doc := range fallbackResults {
		docByID[doc.ID] = doc
	}
	results := make([]*schema.Document, 0, len(fallbackResults))
	seen := make(map[string]bool, len(fallbackResults))
	for rank, id := range orderedIDs {
		doc, exists := docByID[id]
		if !exists || seen[id] {
			continue
		}
		seen[id] = true
		score := 1.0 / float64(rank+1)
		doc.WithScore(score)
		doc.MetaData["rerank_mode"] = "llm"
		doc.MetaData["rerank_score"] = score
		results = append(results, doc)
	}
	for _, doc := range fallbackResults {
		if !seen[doc.ID] {
			results = append(results, doc)
		}
	}
	return limit(results, topK), nil
}

func parseOrderedIDs(content string) ([]string, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("rerank response does not contain JSON")
	}
	var payload struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &payload); err != nil {
		return nil, err
	}
	return payload.OrderedIDs, nil
}

func truncateRunes(content string, limit int) string {
	runes := []rune(content)
	if len(runes) <= limit {
		return content
	}
	return string(runes[:limit])
}
