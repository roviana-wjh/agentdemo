package blog_tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type GetRecentCommentsInput struct {
	ArticleID uint `json:"article_id,omitempty" jsonschema:"description=Optional published article ID; omit to query comments across all published articles"`
	Limit     int  `json:"limit,omitempty" jsonschema:"description=Maximum number of recent comments to return between 1 and 50"`
}

func newGetRecentCommentsTool(client *Client) (tool.BaseTool, error) {
	created, err := toolutils.InferTool(
		"get_recent_comments",
		"Get recent comments for one published article or across all published articles. User passwords and private data are never returned.",
		func(ctx context.Context, input GetRecentCommentsInput) (GetRecentCommentsResult, error) {
			return client.GetRecentComments(ctx, input.ArticleID, normalizeLimit(input.Limit, 10, 50))
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create get_recent_comments tool: %w", err)
	}
	return created, nil
}
