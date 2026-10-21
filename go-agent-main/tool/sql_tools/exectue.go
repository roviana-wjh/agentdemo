package sql_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go-agent/config"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	cachedMCPTools []tool.BaseTool
	mcpOnce        sync.Once
	mcpInitErr     error
)

// InitMCPTools initializes the MCP connection and caches tool instances.
func InitMCPTools(ctx context.Context) error {
	mcpOnce.Do(func() {
		cachedMCPTools, mcpInitErr = connectMCP(ctx)
	})
	return mcpInitErr
}

func connectMCP(ctx context.Context) ([]tool.BaseTool, error) {
	cli := mcp.NewClient(&mcp.Implementation{
		Name:    "go-agent-client",
		Version: "1.0.0",
	}, nil)

	cmd := exec.Command(npxCommand(), "-y", "mcp-server-mysql")
	cmd.Env = append(os.Environ(),
		"MYSQL_HOST="+config.Cfg.MySQLConf.Host,
		"MYSQL_PORT="+config.Cfg.MySQLConf.Port,
		"MYSQL_USER="+config.Cfg.MySQLConf.Username,
		"MYSQL_PASS="+config.Cfg.MySQLConf.Password,
		"MYSQL_DB="+config.Cfg.MySQLConf.Database,
	)

	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	session, err := cli.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("MCP client connection failed: %w", err)
	}

	return officialmcp.GetTools(ctx, &officialmcp.Config{
		Cli:          session,
		ToolNameList: []string{"mysql_query", "list_tables", "describe_table"},
	})
}

func npxCommand() string {
	if runtime.GOOS == "windows" {
		return "npx.cmd"
	}
	return "npx"
}

// GetMCPTool returns the cached MCP tools.
func GetMCPTool(ctx context.Context) ([]tool.BaseTool, error) {
	if cachedMCPTools == nil {
		return nil, fmt.Errorf("MCP tools are not initialized; call InitMCPTools first")
	}
	return cachedMCPTools, nil
}

func SQLExecute(ctx context.Context, sql string) (string, error) {
	mcpTool, err := GetMCPTool(ctx)
	if err != nil {
		return "", err
	}

	var targetTool tool.InvokableTool
	var toolNames []string
	for _, t := range mcpTool {
		if invokable, ok := t.(tool.InvokableTool); ok {
			info, _ := invokable.Info(ctx)
			toolNames = append(toolNames, info.Name)
			if info.Name == "mysql_query" {
				targetTool = invokable
				break
			}
		}
	}

	if targetTool == nil {
		return "", fmt.Errorf("mysql_query tool not found; available tools: %v", toolNames)
	}

	params := map[string]string{"sql": sql}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to serialize MCP params: %w", err)
	}

	result, err := targetTool.InvokableRun(ctx, string(paramsJSON))
	if err != nil {
		return "", fmt.Errorf("MCP tool execution failed: %w", err)
	}

	_ = compose.ProcessState[*SQLState](ctx, func(ctx context.Context, state *SQLState) error {
		state.Result = result
		return nil
	})

	return result, nil
}
