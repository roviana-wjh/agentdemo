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

type ArticleSummaryFlow struct {
	model  model.BaseChatModel
	detail tool.InvokableTool
}

func (f *ArticleSummaryFlow) Stream(ctx context.Context, articleID uint) (*schema.StreamReader[*schema.Message], error) {
	if articleID == 0 {
		return nil, fmt.Errorf("article_id must be greater than zero")
	}
	input, _ := json.Marshal(blog_tools.GetArticleDetailInput{ArticleID: articleID})
	article, err := f.detail.InvokableRun(ctx, string(input))
	if err != nil {
		return nil, fmt.Errorf("get article for summary: %w", err)
	}
	return f.model.Stream(ctx, []*schema.Message{
		schema.SystemMessage("你是中文技术文章编辑。只根据给定文章生成忠实摘要，不补充文章外事实。输出 Markdown，依次包含：一句话结论、3-5 个要点、适合读者。"),
		schema.UserMessage("请总结以下已发布文章：\n" + article),
	})
}
