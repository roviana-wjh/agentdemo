package blog_tools

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go-agent/config"

	"github.com/cloudwego/eino/components/tool"
)

// GetTools builds the read-only Blog domain tools from application config.
func GetTools() ([]tool.BaseTool, error) {
	if config.Cfg == nil {
		return nil, fmt.Errorf("config is not initialized")
	}
	conf := config.Cfg.BlogAPIConf
	return NewTools(conf.BaseURL, conf.InternalToken, conf.Timeout)
}

func NewTools(baseURL, internalToken string, timeout time.Duration) ([]tool.BaseTool, error) {
	if strings.TrimSpace(internalToken) == "" {
		return nil, fmt.Errorf("blog internal token is not configured")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client, err := NewClient(baseURL, internalToken, &http.Client{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	return NewToolsWithClient(client)
}

func NewToolsWithClient(client *Client) ([]tool.BaseTool, error) {
	if client == nil {
		return nil, fmt.Errorf("blog API client is nil")
	}
	builders := []func(*Client) (tool.BaseTool, error){
		newSearchArticleTool,
		newGetArticleDetailTool,
		newGetHotArticlesTool,
		newGetRecentCommentsTool,
		newGetHotTagsTool,
		newGetBlogStatisticsTool,
	}
	tools := make([]tool.BaseTool, 0, len(builders)+1)
	for _, build := range builders {
		created, err := build(client)
		if err != nil {
			return nil, err
		}
		tools = append(tools, created)
	}

	// Add skill_view tool if skill store is available
	if globalSkillStore != nil {
		skillTool, err := newSkillViewTool()
		if err == nil {
			tools = append(tools, skillTool)
		}
	}

	return tools, nil
}

func normalizeLimit(value, fallback, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}
