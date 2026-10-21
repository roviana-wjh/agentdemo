package flow

import (
	"context"
	"fmt"
	"go-agent/config"
	"go-agent/model/chat_model"
	"go-agent/tool"
	"go-agent/tool/sql_tools"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type FinalGraphRequest struct {
	Query     string `json:"query" binding:"required"`
	SessionID string `json:"session_id,omitempty"`
	SQL       string `json:"sql,omitempty"`
	Docs      string `json:"docs,omitempty"`
}

const (
	Trans_List   = "Trans_List"
	Intent_Model = "Intent_Model"
	React        = "React"
	Chat         = "Chat"
	ChatToEnd    = "ChatToEnd"
	ToToolCall   = "ToToolCall"
	MCP          = "MCP"
)

func init() {
	schema.Register[*FinalGraphRequest]()
}

var (
	cachedFinalGraph  compose.Runnable[FinalGraphRequest, []*schema.Message]
	finalGraphOnce    sync.Once
	finalGraphInitErr error
)

func InitFinalGraph(ctx context.Context, store compose.CheckPointStore) error {
	finalGraphOnce.Do(func() {
		cachedFinalGraph, finalGraphInitErr = buildFinalGraph(ctx, store)
	})
	return finalGraphInitErr
}

func GetFinalGraph() (compose.Runnable[FinalGraphRequest, []*schema.Message], error) {
	if cachedFinalGraph == nil {
		return nil, fmt.Errorf("FinalGraph is not initialized; call InitFinalGraph first")
	}
	return cachedFinalGraph, nil
}

func buildFinalGraph(ctx context.Context, store compose.CheckPointStore) (compose.Runnable[FinalGraphRequest, []*schema.Message], error) {
	g := compose.NewGraph[FinalGraphRequest, []*schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) *FinalGraphRequest {
			return &FinalGraphRequest{}
		}),
	)

	_ = g.AddLambdaNode(Intent_Model, compose.InvokableLambda(func(ctx context.Context, input FinalGraphRequest) (*schema.Message, error) {
		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			*state = input
			return nil
		})

		if isSQLIntent(input.Query) {
			return schema.AssistantMessage("SQL", nil), nil
		}

		intentTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage("You are an intent classifier. If the user asks for database queries, statistics, reports, table creation, schema changes, inserts, updates, deletes, or any SQL/DDL/DML operation, answer exactly SQL. Otherwise answer exactly Chat. Do not output anything else."),
			schema.UserMessage("{query}"),
		)
		cm, err := chat_model.GetChatModel(context.Background(), config.Cfg.IntentModelType)
		if err != nil {
			return nil, err
		}
		output, err := intentTemp.Format(ctx, map[string]any{
			"query": input.Query,
		})
		if err != nil {
			return nil, err
		}
		return cm.Generate(ctx, output)
	}))

	react, err := BuildReactGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("build React graph failed: %w", err)
	}
	_ = g.AddGraphNode(React, react, compose.WithStatePreHandler(func(ctx context.Context, in []*schema.Message, state *FinalGraphRequest) ([]*schema.Message, error) {
		return []*schema.Message{schema.UserMessage(state.Query)}, nil
	}))

	chat, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatModelNode(Chat, chat, compose.WithStatePreHandler(func(ctx context.Context, in []*schema.Message, state *FinalGraphRequest) ([]*schema.Message, error) {
		return []*schema.Message{schema.UserMessage(state.Query)}, nil
	}))

	_ = g.AddLambdaNode(ChatToEnd, compose.InvokableLambda(tool.MsgToMsgs))
	_ = g.AddLambdaNode(Trans_List, compose.InvokableLambda(tool.MsgToMsgs))

	_ = g.AddBranch(Trans_List, compose.NewGraphBranch(func(ctx context.Context, input []*schema.Message) (endNode string, err error) {
		content := strings.ToUpper(input[len(input)-1].Content)
		if strings.Contains(content, "SQL") {
			return React, nil
		}
		return Chat, nil
	}, map[string]bool{
		React: true,
		Chat:  true,
	}))

	_ = g.AddLambdaNode(ToToolCall, compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (*schema.Message, error) {
		msg, err := tool.MsgsToMsg(ctx, input)
		if err != nil {
			return nil, err
		}
		msg.Content = strings.TrimPrefix(msg.Content, "YES: ")
		return tool.MsgToSQLToolCall(ctx, msg)
	}))

	tools, err := sql_tools.GetMCPTool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get MCP tools failed: %w", err)
	}
	mcpTool, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: tools,
	})
	if err != nil {
		return nil, err
	}
	_ = g.AddToolsNode(MCP, mcpTool)

	_ = g.AddEdge(compose.START, Intent_Model)
	_ = g.AddEdge(Intent_Model, Trans_List)
	_ = g.AddEdge(React, ToToolCall)
	_ = g.AddEdge(ToToolCall, MCP)
	_ = g.AddEdge(MCP, compose.END)
	_ = g.AddEdge(Chat, ChatToEnd)
	_ = g.AddEdge(ChatToEnd, compose.END)

	return g.Compile(ctx, compose.WithCheckPointStore(store))
}

func isSQLIntent(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	keywords := []string{
		"sql", "select", "insert", "update", "delete", "create table", "drop table", "alter table",
		"truncate", "describe", "show tables", "建表", "创建表", "创建一张", "新建表", "删表", "删除表",
		"改表", "修改表", "插入", "写入", "更新", "删除数据", "查询", "统计", "表结构", "字段",
	}
	for _, keyword := range keywords {
		if strings.Contains(q, keyword) {
			return true
		}
	}
	return false
}
