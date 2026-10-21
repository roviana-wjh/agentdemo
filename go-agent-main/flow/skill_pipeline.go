package flow

import (
	"context"
	"log"

	"go-agent/tool/skill"
	"go-agent/tool/trace"
)

// 技能学习 pipeline 的包级依赖。由 main.go 在初始化时通过 SetSkillPipeline 注入。
// 未注入时（MySQL 不可用或未启用），extractAndLearnSkill 静默跳过。
var (
	skillExtractor *skill.SkillExtractor
	skillStore     skill.SkillStore
)

// SetSkillPipeline 注入技能学习依赖。传 nil 表示禁用技能学习。
func SetSkillPipeline(extractor *skill.SkillExtractor, store skill.SkillStore) {
	skillExtractor = extractor
	skillStore = store
}

// skillPipelineEnabled 报告技能学习是否已启用。
func skillPipelineEnabled() bool {
	return skillExtractor != nil && skillStore != nil
}

// triggerSkillLearning 异步触发技能学习，从成功的工具调用中抽取技能。
func triggerSkillLearning(ctx context.Context, query string, toolCalls []skill.ToolCallRecord) {
	go extractAndLearnSkill(context.Background(), query, toolCalls, 3)
}

// extractAndLearnSkill 从成功的工具调用序列中抽取技能并入库。
// 仅当工具调用次数 ≥ 阈值且无错误时触发。在后台 goroutine 中执行。
func extractAndLearnSkill(ctx context.Context, query string, toolCalls []skill.ToolCallRecord, minToolCalls int) {
	// Generate trace ID for this skill learning pipeline
	traceID := trace.GenerateTraceID()
	ctx = trace.WithTraceID(ctx, traceID)

	log.Printf("[SkillPipeline] Starting pipeline for query: %q, total_tool_calls: %d, trace_id: %s",
		query, len(toolCalls), traceID)

	if !skillPipelineEnabled() {
		log.Printf("[SkillPipeline] Pipeline disabled, skipping")
		return
	}

	// 过滤出成功的工具调用
	var successCalls []skill.ToolCallRecord
	for _, tc := range toolCalls {
		if tc.Error == nil {
			successCalls = append(successCalls, tc)
		}
	}

	// 触发条件：至少 minToolCalls 次成功调用
	if len(successCalls) < minToolCalls {
		log.Printf("[SkillPipeline] Below threshold: successful=%d, required=%d", len(successCalls), minToolCalls)
		return
	}

	log.Printf("[SkillPipeline] Triggering extraction: successful_calls=%d", len(successCalls))

	// 调用抽取器生成技能
	learned, err := skillExtractor.Extract(ctx, query, successCalls)
	if err != nil {
		log.Printf("warning: skill extraction failed for query %q: %v", query, err)
		return
	}

	if learned == nil || learned.Name == "" {
		// 抽取器认为不值得学习
		log.Printf("[SkillPipeline] No skill learned from this interaction")
		return
	}

	log.Printf("[SkillPipeline] Skill extracted: '%s', saving to store (trace_id: %s)", learned.Name, traceID)

	// 入库（自动去重）
	if err := skillStore.Save(ctx, learned); err != nil {
		trace.LogSkillSave(traceID, learned.Name, false, err)
		log.Printf("warning: skill save failed for '%s': %v", learned.Name, err)
		return
	}

	trace.LogSkillSave(traceID, learned.Name, true, nil)
	log.Printf("info: learned new skill '%s' from %d tool calls (trace_id: %s)",
		learned.Name, len(successCalls), traceID)
}

// loadSkillSummaries 加载技能索引（name+description），供注入 prompt 使用。
// 未启用或出错时返回 nil，不影响主流程。
func loadSkillSummaries(ctx context.Context, limit int) []*skill.SkillSummary {
	if !skillPipelineEnabled() {
		return nil
	}
	summaries, err := skillStore.List(ctx, limit)
	if err != nil {
		log.Printf("warning: load skill summaries failed: %v", err)
		return nil
	}
	return summaries
}

// buildSkillsMessage 把技能索引渲染成一条 SystemMessage；无技能时返回 nil。
func buildSkillsMessage(summaries []*skill.SkillSummary) string {
	if len(summaries) == 0 {
		return ""
	}
	text := "已学习的技能列表（可用 skill_view 工具查看完整流程）：\n"
	for _, s := range summaries {
		text += "- " + s.Name + ": " + s.Description + "\n"
	}
	return text
}
