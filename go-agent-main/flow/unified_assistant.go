package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const (
	IntentKnowledgeRAG = "knowledge_rag"
	IntentBlogTool     = "blog_tool"
)

type UnifiedChatInput struct {
	Query     string `json:"query"`
	SessionID string `json:"session_id,omitempty"`
}

type IntentResult struct {
	Intent string `json:"intent"`
	Reason string `json:"reason,omitempty"`
}

type UnifiedAssistant struct {
	model model.BaseChatModel
}

var (
	unifiedMu     sync.RWMutex
	cachedUnified *UnifiedAssistant
)

func InitUnifiedAssistant(chatModel model.BaseChatModel) error {
	if chatModel == nil {
		return fmt.Errorf("unified assistant model is nil")
	}
	unifiedMu.Lock()
	cachedUnified = &UnifiedAssistant{model: chatModel}
	unifiedMu.Unlock()
	return nil
}

func GetUnifiedAssistant() (*UnifiedAssistant, error) {
	unifiedMu.RLock()
	defer unifiedMu.RUnlock()
	if cachedUnified == nil {
		return nil, fmt.Errorf("unified assistant is not initialized")
	}
	return cachedUnified, nil
}

func (a *UnifiedAssistant) Classify(ctx context.Context, query string) (IntentResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return IntentResult{}, fmt.Errorf("query must not be empty")
	}
	if a == nil || a.model == nil {
		return IntentResult{}, fmt.Errorf("unified assistant model is nil")
	}
	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(`你是博客助手的意图分类器。只允许输出 JSON，不要输出 Markdown。
knowledge_rag：用户在询问博客文章中的知识、技术解释、文章内容或需要基于知识库检索。
blog_tool：用户在查询博客业务数据，例如热门文章、文章详情、评论、标签、博客统计。
格式：{"intent":"knowledge_rag|blog_tool","reason":"简短原因"}`),
		schema.UserMessage(query),
	})
	if err == nil && response != nil {
		var result IntentResult
		if json.Unmarshal([]byte(extractJSONObject(response.Content)), &result) == nil {
			result.Intent = normalizeIntent(result.Intent)
			result.Reason = strings.TrimSpace(result.Reason)
			if result.Intent == IntentKnowledgeRAG || result.Intent == IntentBlogTool {
				return result, nil
			}
		}
	}
	// Classification failure is fail-safe: factual/technical questions use RAG,
	// while explicit business-data words use the tool agent.
	return IntentResult{Intent: fallbackIntent(query), Reason: "fallback classification"}, nil
}

func normalizeIntent(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case IntentBlogTool:
		return IntentBlogTool
	case IntentKnowledgeRAG:
		return IntentKnowledgeRAG
	default:
		return ""
	}
}

func fallbackIntent(query string) string {
	for _, keyword := range []string{
		"热门", "最热", "评论", "留言", "回复", "标签", "统计", "排行", "排行榜",
		"文章详情", "最近文章", "最新文章", "最近发布", "最近更新", "阅读量", "浏览量", "点击量",
	} {
		if strings.Contains(query, keyword) {
			return IntentBlogTool
		}
	}
	return IntentKnowledgeRAG
}
