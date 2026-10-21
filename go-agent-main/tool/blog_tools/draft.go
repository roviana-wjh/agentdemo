package blog_tools

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go-agent/config"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type CreateArticleDraftInput struct {
	AuthorID uint   `json:"author_id" jsonschema:"required,description=Authenticated numeric author ID supplied by the Blog gateway"`
	Title    string `json:"title" jsonschema:"required,description=Draft article title"`
	Content  string `json:"content" jsonschema:"required,description=Draft article body in Markdown"`
	Image    string `json:"image,omitempty" jsonschema:"description=Optional cover image URL"`
}

type UpdateArticleDraftInput struct {
	AuthorID  uint   `json:"author_id" jsonschema:"required,description=Authenticated numeric author ID supplied by the Blog gateway"`
	ArticleID uint   `json:"article_id" jsonschema:"required,description=Numeric ID of the author's existing draft"`
	Title     string `json:"title" jsonschema:"required,description=Updated draft title"`
	Content   string `json:"content" jsonschema:"required,description=Updated draft body in Markdown"`
	Image     string `json:"image,omitempty" jsonschema:"description=Optional cover image URL"`
}

type DraftResult struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Version uint   `json:"version"`
	Status  string `json:"status"`
}

func NewDraftToolsFromConfig() ([]tool.BaseTool, error) {
	if config.Cfg == nil {
		return nil, fmt.Errorf("config is not initialized")
	}
	conf := config.Cfg.BlogAPIConf
	if strings.TrimSpace(conf.InternalToken) == "" {
		return nil, fmt.Errorf("blog internal token is not configured")
	}
	timeout := conf.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client, err := NewClient(conf.BaseURL, conf.InternalToken, &http.Client{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	return NewDraftToolsWithClient(client)
}

func NewDraftToolsWithClient(client *Client) ([]tool.BaseTool, error) {
	if client == nil {
		return nil, fmt.Errorf("blog API client is nil")
	}
	createTool, err := toolutils.InferTool(
		"create_article_draft",
		"Create an unpublished blog draft after explicit user confirmation. This tool never publishes articles.",
		func(ctx context.Context, input CreateArticleDraftInput) (DraftResult, error) {
			if err := validateDraft(input.AuthorID, input.Title, input.Content); err != nil {
				return DraftResult{}, err
			}
			return client.CreateArticleDraft(ctx, input)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create create_article_draft tool: %w", err)
	}
	updateTool, err := toolutils.InferTool(
		"update_article_draft",
		"Update an existing unpublished draft owned by the authenticated author after explicit confirmation. This tool never publishes articles.",
		func(ctx context.Context, input UpdateArticleDraftInput) (DraftResult, error) {
			if input.ArticleID == 0 {
				return DraftResult{}, fmt.Errorf("article_id must be greater than zero")
			}
			if err := validateDraft(input.AuthorID, input.Title, input.Content); err != nil {
				return DraftResult{}, err
			}
			return client.UpdateArticleDraft(ctx, input)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create update_article_draft tool: %w", err)
	}
	return []tool.BaseTool{createTool, updateTool}, nil
}

func (c *Client) CreateArticleDraft(ctx context.Context, input CreateArticleDraftInput) (DraftResult, error) {
	var envelope struct {
		Data DraftResult `json:"data"`
	}
	err := c.post(ctx, "/internal/v1/tools/create_article_draft", input, &envelope)
	return envelope.Data, err
}

func (c *Client) UpdateArticleDraft(ctx context.Context, input UpdateArticleDraftInput) (DraftResult, error) {
	var envelope struct {
		Data DraftResult `json:"data"`
	}
	err := c.post(ctx, "/internal/v1/tools/update_article_draft", input, &envelope)
	return envelope.Data, err
}

func validateDraft(authorID uint, title, content string) error {
	if authorID == 0 {
		return fmt.Errorf("author_id must be greater than zero")
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title must not be empty")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content must not be empty")
	}
	return nil
}
