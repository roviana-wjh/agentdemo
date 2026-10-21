package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"go-agent/tool/trace"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// SkillExtractor extracts reusable skills from successful tool call sequences using LLM.
type SkillExtractor struct {
	Model model.BaseChatModel
}

// NewSkillExtractor creates a skill extractor with the given LLM.
func NewSkillExtractor(m model.BaseChatModel) *SkillExtractor {
	return &SkillExtractor{Model: m}
}

// ToolCallRecord represents a single tool invocation (matches flow.ToolCallRecord).
type ToolCallRecord struct {
	Name   string
	Input  string
	Output string
	Error  error
}

// Extract analyzes a successful tool call sequence and generates a skill if worthwhile.
// Returns nil if the sequence is too simple, failed, or not worth preserving.
func (e *SkillExtractor) Extract(ctx context.Context, userQuery string, records []ToolCallRecord) (*Skill, error) {
	if e == nil || e.Model == nil {
		return nil, nil
	}

	traceID := trace.GetTraceID(ctx)
	startTime := time.Now()

	// Log extraction start
	trace.LogExtractionStart(traceID, userQuery, len(records))
	log.Printf("[SkillExtraction] Starting extraction for query: %q, tool_calls: %d, trace_id: %s",
		userQuery, len(records), traceID)

	// Trigger threshold: only learn from sequences with 3+ tool calls
	if len(records) < 3 {
		trace.LogExtractionSkipped(traceID, "below_threshold", len(records))
		log.Printf("[SkillExtraction] Skipped: tool call count %d below threshold 3", len(records))
		return nil, nil
	}

	// Filter out failed tool calls
	var successfulRecords []ToolCallRecord
	for _, r := range records {
		if r.Error == nil {
			successfulRecords = append(successfulRecords, r)
		}
	}

	if len(successfulRecords) < 3 {
		trace.LogExtractionSkipped(traceID, "insufficient_successful_calls", len(successfulRecords))
		log.Printf("[SkillExtraction] Skipped: successful calls %d/%d below threshold", len(successfulRecords), len(records))
		return nil, nil
	}

	toolChain := formatToolChain(successfulRecords)
	if strings.TrimSpace(toolChain) == "" {
		return nil, nil
	}

	prompt := fmt.Sprintf(`从成功的工具调用链中抽取可复用的技能。输出JSON(无Markdown):
{
  "name": "kebab-case名称",
  "description": "一句话功能描述",
  "trigger_tags": "触发关键词,逗号分隔",
  "procedure": "详细流程说明,供AI理解执行",
  "tools_used": "工具名,逗号分隔"
}

用户问题: %s
工具调用链:
%s

如不值得保存(太简单/失败/一次性)输出null。`, userQuery, toolChain)

	// LLM call will be automatically traced by global LangSmith handler if configured
	resp, err := e.Model.Generate(ctx, []*schema.Message{
		schema.UserMessage(prompt),
	})
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		trace.LogExtractionFailed(traceID, userQuery, duration, err)
		log.Printf("[SkillExtraction] LLM generation failed: %v", err)
		return nil, fmt.Errorf("skill extraction generate: %w", err)
	}

	jsonStr := extractJSONObject(resp.Content)
	if strings.TrimSpace(jsonStr) == "null" || jsonStr == "{}" {
		trace.LogExtractionSkipped(traceID, "llm_declined", len(successfulRecords))
		log.Printf("[SkillExtraction] LLM decided not to extract skill")
		return nil, nil
	}

	var raw struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		TriggerTags string `json:"trigger_tags"`
		Procedure   string `json:"procedure"`
		ToolsUsed   string `json:"tools_used"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		trace.LogExtractionSkipped(traceID, "json_parse_failed", len(successfulRecords))
		log.Printf("[SkillExtraction] JSON parse failed: %v, content: %s", err, jsonStr)
		// Parse failure treated as "not worth saving"
		return nil, nil
	}

	if raw.Name == "" || raw.Procedure == "" {
		trace.LogExtractionSkipped(traceID, "incomplete_data", len(successfulRecords))
		log.Printf("[SkillExtraction] Extracted data incomplete: name=%q, procedure_len=%d", raw.Name, len(raw.Procedure))
		return nil, nil
	}

	skill := &Skill{
		Name:        raw.Name,
		Description: raw.Description,
		TriggerTags: raw.TriggerTags,
		Procedure:   raw.Procedure,
		ToolsUsed:   raw.ToolsUsed,
	}

	// Log extraction success with metadata
	trace.LogExtractionSuccess(traceID, skill.Name, duration, map[string]interface{}{
		"tools_used":   skill.ToolsUsed,
		"description":  skill.Description,
		"trigger_tags": skill.TriggerTags,
	})
	log.Printf("[SkillExtraction] Success: extracted skill '%s' (tools: %s, duration: %dms, trace_id: %s)",
		skill.Name, skill.ToolsUsed, duration, traceID)

	return skill, nil
}

// formatToolChain renders tool call records as a numbered list for LLM consumption.
func formatToolChain(records []ToolCallRecord) string {
	var b strings.Builder
	for i, r := range records {
		b.WriteString(fmt.Sprintf("%d. %s(%s) -> %s\n",
			i+1, r.Name, truncate(r.Input, 100), truncate(r.Output, 200)))
	}
	return b.String()
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractJSONObject extracts JSON object from potentially Markdown-wrapped text.
func extractJSONObject(text string) string {
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		return text[start : end+1]
	}
	return "{}"
}
