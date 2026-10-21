package flow

import (
	"context"
	"fmt"
	"io"
	"sync"

	"go-agent/tool/blog_tools"
	"go-agent/tool/skill"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

const blogAssistantPersona = `你是博客领域助手。你只能通过提供的博客领域 Tool 查询文章、热度和评论，不能生成或执行 SQL，也不能猜测数据库内容。
仅回答 Tool 返回的数据能够支持的事实。找不到数据时明确说明。涉及文章时尽量给出文章 ID 和标题；不得泄露内部鉴权信息。`

type BlogAssistant struct {
	agent *react.Agent
}

var (
	blogAssistantMu     sync.RWMutex
	cachedBlogAssistant *BlogAssistant
)

func InitBlogAssistant(ctx context.Context, chatModel model.BaseChatModel) error {
	tools, err := blog_tools.GetTools()
	if err != nil {
		return fmt.Errorf("load blog domain tools: %w", err)
	}
	assistant, err := NewBlogAssistant(ctx, chatModel, tools)
	if err != nil {
		return err
	}
	blogAssistantMu.Lock()
	cachedBlogAssistant = assistant
	blogAssistantMu.Unlock()
	return nil
}

func NewBlogAssistant(ctx context.Context, chatModel model.BaseChatModel, toolsConfig []tool.BaseTool) (*BlogAssistant, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("blog assistant chat model is nil")
	}
	config := &react.AgentConfig{
		ToolsConfig:     compose.ToolsNodeConfig{Tools: toolsConfig},
		MessageModifier: react.NewPersonaModifier(blogAssistantPersona),
		MaxStep:         12,
		GraphName:       "BlogAssistant",
	}
	if toolCallingModel, ok := chatModel.(model.ToolCallingChatModel); ok {
		config.ToolCallingModel = toolCallingModel
	} else if legacyModel, ok := chatModel.(model.ChatModel); ok {
		config.Model = legacyModel
	} else {
		return nil, fmt.Errorf("configured chat model does not support tool calling")
	}
	agent, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("build blog assistant: %w", err)
	}
	return &BlogAssistant{agent: agent}, nil
}

func GetBlogAssistant() (*BlogAssistant, error) {
	blogAssistantMu.RLock()
	defer blogAssistantMu.RUnlock()
	if cachedBlogAssistant == nil {
		return nil, fmt.Errorf("blog assistant is not initialized")
	}
	return cachedBlogAssistant, nil
}

// GenerateWithSkillTracking executes the agent with tool call tracking enabled.
// Returns the final message, tool call records, and any error.
func (a *BlogAssistant) GenerateWithSkillTracking(ctx context.Context, messages []*schema.Message) (*schema.Message, []skill.ToolCallRecord, error) {
	collector, handler := NewToolCallCollector()
	msg, err := a.agent.Generate(ctx, messages,
		agent.WithComposeOptions(compose.WithCallbacks(handler)))
	return msg, collector.GetRecords(), err
}

func (a *BlogAssistant) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	msg, records, err := a.GenerateWithSkillTracking(ctx, messages)
	if err == nil && len(records) >= 3 {
		userQuery := extractUserQuery(messages)
		triggerSkillLearning(ctx, userQuery, records)
	}
	return msg, err
}

func (a *BlogAssistant) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	collector, handler := NewToolCallCollector()
	stream, err := a.agent.Stream(ctx, messages,
		agent.WithComposeOptions(compose.WithCallbacks(handler)))
	if err != nil {
		return nil, err
	}

	// 包装流，在流结束时触发技能学习
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					// 流结束，检查是否需要触发技能学习
					records := collector.GetRecords()
					if len(records) >= 3 {
						userQuery := extractUserQuery(messages)
						triggerSkillLearning(ctx, userQuery, records)
					}
				}
				return
			}
			if ok := writer.Send(msg, nil); !ok {
				return
			}
		}
	}()
	return reader, nil
}

// extractUserQuery extracts the last user message content for skill extraction.
func extractUserQuery(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == schema.User {
			return messages[i].Content
		}
	}
	return ""
}
