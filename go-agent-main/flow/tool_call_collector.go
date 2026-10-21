package flow

import (
	"context"
	"sync"

	"go-agent/tool/skill"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	ub "github.com/cloudwego/eino/utils/callbacks"
)

// ToolCallCollector captures tool invocations during agent execution.
type ToolCallCollector struct {
	records []skill.ToolCallRecord
	mu      sync.Mutex
}

// NewToolCallCollector creates a collector and returns both the collector and its callback handler.
func NewToolCallCollector() (*ToolCallCollector, callbacks.Handler) {
	collector := &ToolCallCollector{}

	toolHandler := &ub.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			collector.mu.Lock()
			collector.records = append(collector.records, skill.ToolCallRecord{
				Name:  info.Name,
				Input: input.ArgumentsInJSON,
			})
			collector.mu.Unlock()
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			collector.mu.Lock()
			if len(collector.records) > 0 {
				collector.records[len(collector.records)-1].Output = output.Response
			}
			collector.mu.Unlock()
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			collector.mu.Lock()
			if len(collector.records) > 0 {
				collector.records[len(collector.records)-1].Error = err
			}
			collector.mu.Unlock()
			return ctx
		},
	}

	handler := ub.NewHandlerHelper().Tool(toolHandler).Handler()
	return collector, handler
}

// GetRecords returns a copy of all captured tool call records.
func (c *ToolCallCollector) GetRecords() []skill.ToolCallRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	copied := make([]skill.ToolCallRecord, len(c.records))
	copy(copied, c.records)
	return copied
}
