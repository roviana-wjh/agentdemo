package blog_tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type SearchArticleInput struct {
	Keyword string `json:"keyword" jsonschema:"required,description=Keywords that must appear in the published article title or content"`
	Limit   int    `json:"limit,omitempty" jsonschema:"description=Maximum number of articles to return between 1 and 20"`
}

type GetArticleDetailInput struct {
	ArticleID uint `json:"article_id" jsonschema:"required,description=Numeric ID of a published article"`
}

func newSearchArticleTool(client *Client) (tool.BaseTool, error) {
	created, err := toolutils.InferTool(
		"search_article",
		"Search published blog articles by business keywords. Returns filtered article summaries without exposing database access.",
		func(ctx context.Context, input SearchArticleInput) (SearchArticleResult, error) {
			keyword := strings.TrimSpace(input.Keyword)
			if keyword == "" {
				return SearchArticleResult{}, fmt.Errorf("keyword must not be empty")
			}
			return client.SearchArticle(ctx, keyword, normalizeLimit(input.Limit, 10, 20))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create search_article tool: %w", err)
	}
	return created, nil
}

func newGetArticleDetailTool(client *Client) (tool.BaseTool, error) {
	created, err := toolutils.InferTool(
		"get_article_detail",
		"Get the safe full details of one published blog article by ID.",
		func(ctx context.Context, input GetArticleDetailInput) (GetArticleDetailResult, error) {
			if input.ArticleID == 0 {
				return GetArticleDetailResult{}, fmt.Errorf("article_id must be greater than zero")
			}
			return client.GetArticleDetail(ctx, input.ArticleID)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create get_article_detail tool: %w", err)
	}
	return created, nil
}
