package flow

import (
	"context"
	"fmt"
	"go-agent/config"
	"go-agent/model/chat_model"
	"go-agent/rag/rag_flow"
	"go-agent/tool"
	"regexp"
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

	retriever, err := rag_flow.BuildRetrieverGraph(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddGraphNode(SQL_Retrieve, retriever)

	_ = g.AddLambdaNode(ToTplVar, compose.InvokableLambda(func(ctx context.Context, input []*schema.Document) (map[string]any, error) {
		var query string
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

	sqlTemp := prompt.FromMessages(schema.FString,
		schema.SystemMessage("You are a MySQL SQL expert. Generate exactly one SQL statement for the user request. Output SQL only, no Markdown and no explanation. If the user explicitly asks to create a table or build a schema, generate CREATE TABLE. If the user explicitly asks to insert, update, delete, alter, or drop data/schema, generate the corresponding DML/DDL. If the user is only querying, viewing, counting, filtering, or reporting data, generate SELECT only. Never turn an explicit CREATE TABLE request into SELECT."),
		schema.UserMessage("Relevant table/schema information:\n{docs}\n\nUser request:\n{query}"),
	)
	_ = g.AddChatTemplateNode(SQL_Tpl, sqlTemp)

	chat, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatModelNode(SQL_Model, chat)

	_ = g.AddLambdaNode(Trans_List, compose.InvokableLambda(tool.MsgToMsgs))

	_ = g.AddLambdaNode(Approve, compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (output *schema.Message, err error) {
		var stateSQL string
		var query string
		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			stateSQL = state.SQL
			query = state.Query
			return nil
		})

		if isResume, hasData, data := compose.GetResumeContext[string](ctx); isResume && hasData {
			if strings.Contains(strings.ToUpper(data), "YES") {
				return schema.AssistantMessage("YES: "+stateSQL, nil), nil
			}
			return schema.AssistantMessage(data, nil), nil
		}

		if input == nil {
			return nil, fmt.Errorf("input is nil")
		}

		if createSQL, ok := deterministicCreateTableSQL(query); ok {
			input.Content = createSQL
		}

		_ = compose.ProcessState[*FinalGraphRequest](ctx, func(ctx context.Context, state *FinalGraphRequest) error {
			state.SQL = input.Content
			return nil
		})

		return nil, compose.Interrupt(ctx, input.Content)
	}))

	_ = g.AddEdge(compose.START, SQL_Retrieve)
	_ = g.AddEdge(SQL_Retrieve, ToTplVar)
	_ = g.AddEdge(ToTplVar, SQL_Tpl)
	_ = g.AddEdge(SQL_Tpl, SQL_Model)
	_ = g.AddEdge(SQL_Model, Approve)
	_ = g.AddEdge(Approve, Trans_List)
	_ = g.AddEdge(Trans_List, compose.END)

	return g, nil
}

func deterministicCreateTableSQL(query string) (string, bool) {
	q := strings.TrimSpace(query)
	lower := strings.ToLower(q)
	if !strings.Contains(lower, "create table") && !strings.Contains(q, "创建") && !strings.Contains(q, "建表") && !strings.Contains(q, "新建") {
		return "", false
	}

	tableName := ""
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)create\s+table\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		regexp.MustCompile(`(?:创建一张|创建|新建|建一张|建)\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*表`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(q)
		if len(matches) > 1 {
			tableName = matches[1]
			break
		}
	}
	if tableName == "" {
		return "", false
	}

	fieldsPart := q
	if idx := strings.Index(q, "包含"); idx >= 0 {
		fieldsPart = q[idx+len("包含"):]
	}
	fieldsPart = strings.ReplaceAll(fieldsPart, "字段", "")
	fieldsPart = strings.ReplaceAll(fieldsPart, "。", "")
	fieldsPart = strings.ReplaceAll(fieldsPart, ".", "")
	fieldTokens := splitFieldTokens(fieldsPart)
	if len(fieldTokens) == 0 {
		return "", false
	}

	columns := make([]string, 0, len(fieldTokens))
	seen := map[string]bool{}
	for _, field := range fieldTokens {
		field = strings.TrimSpace(field)
		field = strings.Trim(field, "`'\" ")
		if field == "" || seen[field] || !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(field) {
			continue
		}
		seen[field] = true
		columns = append(columns, columnDefinition(field))
	}
	if len(columns) == 0 {
		return "", false
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n  %s\n);", tableName, strings.Join(columns, ",\n  ")), true
}

func splitFieldTokens(fieldsPart string) []string {
	replacer := strings.NewReplacer("、", ",", "，", ",", ";", ",", "；", ",", "和", ",", "以及", ",")
	fieldsPart = replacer.Replace(fieldsPart)
	parts := strings.Split(fieldsPart, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func columnDefinition(field string) string {
	switch strings.ToLower(field) {
	case "id":
		return "`id` INT PRIMARY KEY AUTO_INCREMENT"
	case "age":
		return "`age` INT"
	case "created_at", "updated_at":
		return fmt.Sprintf("`%s` TIMESTAMP DEFAULT CURRENT_TIMESTAMP", field)
	case "name":
		return "`name` VARCHAR(100) NOT NULL"
	case "department":
		return "`department` VARCHAR(100)"
	default:
		return fmt.Sprintf("`%s` VARCHAR(255)", field)
	}
}
