package trace

import (
	"encoding/json"
	"log"
	"time"
)

// SkillEvent represents a structured log event for skill learning system.
type SkillEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	TraceID     string                 `json:"trace_id,omitempty"`
	EventType   string                 `json:"event_type"` // extraction, save, cleanup, usage
	Status      string                 `json:"status"`     // success, failed, skipped
	SkillName   string                 `json:"skill_name,omitempty"`
	Query       string                 `json:"query,omitempty"`
	ToolCalls   int                    `json:"tool_calls,omitempty"`
	Duration    int64                  `json:"duration_ms,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// LogSkillEvent outputs a structured JSON log for skill learning events.
// These logs can be collected by Promtail/Fluentd and visualized in Grafana/Kibana.
func LogSkillEvent(event SkillEvent) {
	event.Timestamp = time.Now()
	if data, err := json.Marshal(event); err == nil {
		log.Printf("[SKILL_EVENT] %s", string(data))
	}
}

// Helper functions for common event types

func LogExtractionStart(traceID, query string, toolCalls int) {
	LogSkillEvent(SkillEvent{
		TraceID:   traceID,
		EventType: "extraction",
		Status:    "started",
		Query:     query,
		ToolCalls: toolCalls,
	})
}

func LogExtractionSkipped(traceID, reason string, toolCalls int) {
	LogSkillEvent(SkillEvent{
		TraceID:   traceID,
		EventType: "extraction",
		Status:    "skipped",
		ToolCalls: toolCalls,
		Metadata:  map[string]interface{}{"reason": reason},
	})
}

func LogExtractionSuccess(traceID, skillName string, duration int64, metadata map[string]interface{}) {
	LogSkillEvent(SkillEvent{
		TraceID:   traceID,
		EventType: "extraction",
		Status:    "success",
		SkillName: skillName,
		Duration:  duration,
		Metadata:  metadata,
	})
}

func LogExtractionFailed(traceID, query string, duration int64, err error) {
	LogSkillEvent(SkillEvent{
		TraceID:   traceID,
		EventType: "extraction",
		Status:    "failed",
		Query:     query,
		Duration:  duration,
		Error:     err.Error(),
	})
}

func LogSkillSave(traceID, skillName string, success bool, err error) {
	status := "success"
	errMsg := ""
	if !success {
		status = "failed"
		if err != nil {
			errMsg = err.Error()
		}
	}
	LogSkillEvent(SkillEvent{
		TraceID:   traceID,
		EventType: "save",
		Status:    status,
		SkillName: skillName,
		Error:     errMsg,
	})
}

func LogSkillUsage(skillName string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	LogSkillEvent(SkillEvent{
		EventType: "usage",
		Status:    status,
		SkillName: skillName,
	})
}

func LogCuratorCleanup(lfuCount, lruCount, totalEvaluated int, deleted []string) {
	LogSkillEvent(SkillEvent{
		EventType: "cleanup",
		Status:    "success",
		Metadata: map[string]interface{}{
			"lfu_deleted":     lfuCount,
			"lru_deleted":     lruCount,
			"total_evaluated": totalEvaluated,
			"deleted_skills":  deleted,
		},
	})
}
