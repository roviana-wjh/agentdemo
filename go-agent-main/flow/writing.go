package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"go-agent/tool/blog_tools"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type WritingRequest struct {
	Topic        string `json:"topic"`
	Requirements string `json:"requirements,omitempty"`
}

type SaveDraftRequest struct {
	AuthorID  uint   `json:"author_id"`
	ArticleID uint   `json:"article_id,omitempty"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Image     string `json:"image,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

type WritingFlow struct {
	model       model.BaseChatModel
	search      tool.InvokableTool
	createDraft tool.InvokableTool
	updateDraft tool.InvokableTool
}

var (
	writingFlowMu     sync.RWMutex
	cachedWritingFlow *WritingFlow
)

func InitWritingFlow(chatModel model.BaseChatModel) error {
	if chatModel == nil {
		return fmt.Errorf("writing model is nil")
	}
	readTools, err := blog_tools.GetTools()
	if err != nil {
		return fmt.Errorf("load blog search tool: %w", err)
	}
	draftTools, err := blog_tools.NewDraftToolsFromConfig()
	if err != nil {
		return fmt.Errorf("load blog draft tools: %w", err)
	}
	search, err := invokableToolByName(context.Background(), readTools, "search_article")
	if err != nil {
		return err
	}
	createDraft, err := invokableToolByName(context.Background(), draftTools, "create_article_draft")
	if err != nil {
		return err
	}
	updateDraft, err := invokableToolByName(context.Background(), draftTools, "update_article_draft")
	if err != nil {
		return err
	}
	writingFlowMu.Lock()
	cachedWritingFlow = &WritingFlow{model: chatModel, search: search, createDraft: createDraft, updateDraft: updateDraft}
	writingFlowMu.Unlock()
	return nil
}

func GetWritingFlow() (*WritingFlow, error) {
	writingFlowMu.RLock()
	defer writingFlowMu.RUnlock()
	if cachedWritingFlow == nil {
		return nil, fmt.Errorf("writing flow is not initialized")
	}
	return cachedWritingFlow, nil
}

func (f *WritingFlow) Stream(ctx context.Context, input WritingRequest) (*schema.StreamReader[*schema.Message], error) {
	topic := strings.TrimSpace(input.Topic)
	if topic == "" {
		return nil, fmt.Errorf("topic must not be empty")
	}
	related := "未检索到相关博客文章。"
	arguments, _ := json.Marshal(blog_tools.SearchArticleInput{Keyword: topic, Limit: 5})
	if result, err := f.search.InvokableRun(ctx, string(arguments)); err == nil {
		related = result
	}

	outlinePrompt := []*schema.Message{
		schema.SystemMessage("你是严谨的中文技术编辑。根据主题、用户要求和站内相关资料，生成一个简洁但有逻辑的文章大纲。只输出 Markdown 大纲，不写正文，不编造资料。"),
		schema.UserMessage(fmt.Sprintf("主题：%s\n用户要求：%s\n站内相关资料：%s", topic, input.Requirements, related)),
	}
	outline, err := f.model.Generate(ctx, outlinePrompt)
	if err != nil {
		return nil, fmt.Errorf("generate writing outline: %w", err)
	}
	articlePrompt := []*schema.Message{
		schema.SystemMessage("你是博客作者。根据给定大纲流式写出完整中文 Markdown 文章。第一行必须是唯一的一级标题；结构清晰、内容具体、避免空话。站内资料只用于背景参考，不得复制隐私信息。不要声称已保存或发布文章。"),
		schema.UserMessage(fmt.Sprintf("主题：%s\n补充要求：%s\n大纲：\n%s\n\n相关资料：\n%s", topic, input.Requirements, outline.Content, related)),
	}
	stream, err := f.model.Stream(ctx, articlePrompt)
	if err != nil {
		return nil, fmt.Errorf("stream markdown article: %w", err)
	}
	return stream, nil
}

func (f *WritingFlow) SaveDraft(ctx context.Context, input SaveDraftRequest) (blog_tools.DraftResult, error) {
	if !input.Confirmed {
		return blog_tools.DraftResult{}, fmt.Errorf("explicit user confirmation is required before saving a draft")
	}
	var target tool.InvokableTool
	var arguments any
	if input.ArticleID == 0 {
		target = f.createDraft
		arguments = blog_tools.CreateArticleDraftInput{AuthorID: input.AuthorID, Title: input.Title, Content: input.Content, Image: input.Image}
	} else {
		target = f.updateDraft
		arguments = blog_tools.UpdateArticleDraftInput{AuthorID: input.AuthorID, ArticleID: input.ArticleID, Title: input.Title, Content: input.Content, Image: input.Image}
	}
	payload, err := json.Marshal(arguments)
	if err != nil {
		return blog_tools.DraftResult{}, err
	}
	output, err := target.InvokableRun(ctx, string(payload))
	if err != nil {
		return blog_tools.DraftResult{}, err
	}
	var result blog_tools.DraftResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return blog_tools.DraftResult{}, fmt.Errorf("decode draft tool result: %w", err)
	}
	return result, nil
}

func invokableToolByName(ctx context.Context, tools []tool.BaseTool, name string) (tool.InvokableTool, error) {
	for _, candidate := range tools {
		info, err := candidate.Info(ctx)
		if err != nil {
			return nil, err
		}
		if info.Name == name {
			invokable, ok := candidate.(tool.InvokableTool)
			if !ok {
				return nil, fmt.Errorf("tool %s is not invokable", name)
			}
			return invokable, nil
		}
	}
	return nil, fmt.Errorf("tool %s not found", name)
}
