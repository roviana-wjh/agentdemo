package flow

import (
	"context"
	"fmt"
	"go-agent/config"
	"go-agent/model/chat_model"
	"go-agent/rag/rag_flow"
	"go-agent/tool"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type SQLFlowState struct {
	History []*schema.Message `json:"history"`
}

const (
	SQL_Retrieve = "SQL_Retrieve"
	ToTplVar     = "ToTplVar"
	SQL_Tpl      = "SQL_Tpl"
	SQL_Model    = "SQL_Model"
	Approve      = "Approve"
)

func init() {
	schema.Register[*SQLFlowState]()
}

func BuildReactGraph(ctx context.Context) (*compose.Graph[[]*schema.Message, []*schema.Message], error) {
	g := compose.NewGraph[[]*schema.Message, []*schema.Message]()

	// RAG 检索：召回行业黑话、表结构信息等
	retriever, err := rag_flow.BuildRetrieverGraph(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddGraphNode(SQL_Retrieve, retriever)

	// 转换：[]*Document -> map[string]any (将检索结果包装为模板变量)
	_ = g.AddLambdaNode(ToTplVar, compose.InvokableLambda(func(ctx context.Context, input []*schema.Document) (map[string]any, error) {
		var query string
		// 从全局 State 获取原始 Query
		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			query = state.Query
			return nil
		})

		docsStr := ""
		for _, d := range input {
			docsStr += d.Content + "\n"
		}

		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			state.Docs = docsStr
			return nil
		})

		return map[string]any{
			"query": query,
			"docs":  docsStr,
		}, nil
	}))

	// SQL 模板节点
	sqlTemp := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个 MySQL SQL 专家。请根据用户当前需求和提供的表结构信息生成 SQL。\n规则：\n1. 只输出 SQL，不要解释，不要 Markdown。\n2. 如果用户是在查询、查看、统计、筛选数据，只能生成一条 SELECT 语句，严禁附带 CREATE/INSERT/UPDATE/DELETE/DDL。\n3. 只有用户明确要求创建表、修改表结构或写入数据时，才允许生成对应 DDL/DML。\n4. 表名和字段名必须来自用户当前需求或已提供表结构，不要自行替换成相似表名。\n5. 如果表结构信息不足，也不要自动建表；优先按用户给出的表名字段生成查询 SQL。"),
		schema.UserMessage("相关表结构：\n{docs}\n\n用户需求：{query}"),
	)
	_ = g.AddChatTemplateNode(SQL_Tpl, sqlTemp)

	// SQL 生成模型 (ChatModel)
	chat, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatModelNode(SQL_Model, chat)

	// 转换节点
	_ = g.AddLambdaNode(Trans_List, compose.InvokableLambda(tool.MsgToMsgs))

	// 用户审批节点
	_ = g.AddLambdaNode(Approve, compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (output *schema.Message, err error) {
		var stateSQL string
		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			stateSQL = state.SQL
			return nil
		})

		if isResume, hasData, data := compose.GetResumeContext[string](ctx); isResume && hasData {
			if strings.Contains(strings.ToUpper(data), "YES") {
				// 如果批准了，返回SQL
				return schema.AssistantMessage("YES: "+stateSQL, nil), nil
			}
			return schema.AssistantMessage(data, nil), nil
		}

		if input == nil {
			return nil, fmt.Errorf("input is nil")
		}

		// 保存SQL到状态中
		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			state.SQL = input.Content
			return nil
		})

		return nil, compose.Interrupt(ctx, input.Content)
	}))

	// 连线
	_ = g.AddEdge(compose.START, SQL_Retrieve)
	_ = g.AddEdge(SQL_Retrieve, ToTplVar)
	_ = g.AddEdge(ToTplVar, SQL_Tpl)
	_ = g.AddEdge(SQL_Tpl, SQL_Model)
	_ = g.AddEdge(SQL_Model, Approve)
	_ = g.AddEdge(Approve, Trans_List)
	_ = g.AddEdge(Trans_List, compose.END)

	return g, nil
}
