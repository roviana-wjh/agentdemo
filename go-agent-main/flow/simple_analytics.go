package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go-agent/tool/blog_tools"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type SimpleAnalyticsFlow struct {
	model       model.BaseChatModel
	hotArticles tool.InvokableTool
	hotTags     tool.InvokableTool
	statistics  tool.InvokableTool
}

type InsightFlows struct {
	ArticleSummary  *ArticleSummaryFlow
	CommentAnalysis *CommentAnalysisFlow
	SimpleAnalytics *SimpleAnalyticsFlow
}

var (
	insightFlowsMu     sync.RWMutex
	cachedInsightFlows *InsightFlows
)

func InitInsightFlows(chatModel model.BaseChatModel) error {
	if chatModel == nil {
		return fmt.Errorf("insight model is nil")
	}
	tools, err := blog_tools.GetTools()
	if err != nil {
		return err
	}
	detail, err := invokableToolByName(context.Background(), tools, "get_article_detail")
	if err != nil {
		return err
	}
	comments, err := invokableToolByName(context.Background(), tools, "get_recent_comments")
	if err != nil {
		return err
	}
	hotArticles, err := invokableToolByName(context.Background(), tools, "get_hot_articles")
	if err != nil {
		return err
	}
	hotTags, err := invokableToolByName(context.Background(), tools, "get_hot_tags")
	if err != nil {
		return err
	}
	statistics, err := invokableToolByName(context.Background(), tools, "get_blog_statistics")
	if err != nil {
		return err
	}
	insightFlowsMu.Lock()
	cachedInsightFlows = &InsightFlows{
		ArticleSummary:  &ArticleSummaryFlow{model: chatModel, detail: detail},
		CommentAnalysis: &CommentAnalysisFlow{model: chatModel, comments: comments},
		SimpleAnalytics: &SimpleAnalyticsFlow{model: chatModel, hotArticles: hotArticles, hotTags: hotTags, statistics: statistics},
	}
	insightFlowsMu.Unlock()
	return nil
}

func GetInsightFlows() (*InsightFlows, error) {
	insightFlowsMu.RLock()
	defer insightFlowsMu.RUnlock()
	if cachedInsightFlows == nil {
		return nil, fmt.Errorf("insight flows are not initialized")
	}
	return cachedInsightFlows, nil
}

func (f *SimpleAnalyticsFlow) Stream(ctx context.Context, question string) (*schema.StreamReader[*schema.Message], error) {
	hotArgs, _ := json.Marshal(blog_tools.GetHotArticlesInput{Limit: 10})
	tagArgs, _ := json.Marshal(blog_tools.GetHotTagsInput{Limit: 10})
	hotArticles, err := f.hotArticles.InvokableRun(ctx, string(hotArgs))
	if err != nil {
		return nil, fmt.Errorf("get hot articles: %w", err)
	}
	hotTags, err := f.hotTags.InvokableRun(ctx, string(tagArgs))
	if err != nil {
		return nil, fmt.Errorf("get hot tags: %w", err)
	}
	statistics, err := f.statistics.InvokableRun(ctx, `{}`)
	if err != nil {
		return nil, fmt.Errorf("get blog statistics: %w", err)
	}
	if question == "" {
		question = "概括当前博客的内容表现和读者互动情况"
	}
	prompt := fmt.Sprintf("用户问题：%s\n\n聚合统计：%s\n\n热门文章：%s\n\n热门标签：%s", question, statistics, hotArticles, hotTags)
	return f.model.Stream(ctx, []*schema.Message{
		schema.SystemMessage("你是博客数据分析助手。只依据给定领域统计数据输出中文 Markdown，突出关键数字、热门内容和可行动建议；不生成 SQL，不推测未提供的数据。"),
		schema.UserMessage(prompt),
	})
}
