package flow

import (
	"context"
	"encoding/json"
	"fmt"

	"go-agent/tool/blog_tools"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type CommentAnalysisFlow struct {
	model    model.BaseChatModel
	comments tool.InvokableTool
}

func (f *CommentAnalysisFlow) Stream(ctx context.Context, articleID uint, limit int) (*schema.StreamReader[*schema.Message], error) {
	if limit <= 0 {
		limit = 50
	}
	input, _ := json.Marshal(blog_tools.GetRecentCommentsInput{ArticleID: articleID, Limit: limit})
	comments, err := f.comments.InvokableRun(ctx, string(input))
	if err != nil {
		return nil, fmt.Errorf("get comments for analysis: %w", err)
	}
	return f.model.Stream(ctx, []*schema.Message{
		schema.SystemMessage("你是评论洞察分析师。仅依据给定评论，输出中文 Markdown。包含：整体情感（正向/中性/负向及大致比例）、主要讨论主题、代表性观点（匿名改写）、可行动建议。评论不足时必须说明样本限制，不得识别或推测用户隐私。"),
		schema.UserMessage("请分析以下已发布文章评论：\n" + comments),
	})
}
