package blog_tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type GetHotArticlesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=Maximum number of popular articles to return between 1 and 20"`
}

type GetHotTagsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=Maximum number of popular tags to return between 1 and 30"`
}

type GetBlogStatisticsInput struct{}

func newGetHotArticlesTool(client *Client) (tool.BaseTool, error) {
	created, err := toolutils.InferTool(
		"get_hot_articles",
		"Get the most popular published blog articles ranked by views.",
		func(ctx context.Context, input GetHotArticlesInput) (GetHotArticlesResult, error) {
			return client.GetHotArticles(ctx, normalizeLimit(input.Limit, 5, 20))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create get_hot_articles tool: %w", err)
	}
	return created, nil
}

func newGetHotTagsTool(client *Client) (tool.BaseTool, error) {
	created, err := toolutils.InferTool(
		"get_hot_tags",
		"Get popular tags used by published blog articles, ranked by article count.",
		func(ctx context.Context, input GetHotTagsInput) (GetHotTagsResult, error) {
			return client.GetHotTags(ctx, normalizeLimit(input.Limit, 10, 30))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create get_hot_tags tool: %w", err)
	}
	return created, nil
}

func newGetBlogStatisticsTool(client *Client) (tool.BaseTool, error) {
	created, err := toolutils.InferTool(
		"get_blog_statistics",
		"Get aggregate published-blog statistics: article count, total views, and comment count.",
		func(ctx context.Context, _ GetBlogStatisticsInput) (BlogStatistics, error) {
			return client.GetBlogStatistics(ctx)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create get_blog_statistics tool: %w", err)
	}
	return created, nil
}
