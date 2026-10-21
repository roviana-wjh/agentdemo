package collaborative_writing

import (
	"time"
)

// Metrics 协作写作的性能指标
type Metrics struct {
	// 基础指标
	TotalTime       time.Duration `json:"total_time_ms"`
	SuccessRate     float64       `json:"success_rate"`

	// 各阶段耗时
	ResearchTime    time.Duration `json:"research_time_ms"`
	ArchitectTime   time.Duration `json:"architect_time_ms"`
	WritingTime     time.Duration `json:"writing_time_ms"`
	ReviewTime      time.Duration `json:"review_time_ms"`
	EditingTime     time.Duration `json:"editing_time_ms"`

	// 重写相关
	RewriteCount    int           `json:"rewrite_count"`
	RewriteTime     time.Duration `json:"rewrite_time_ms"`
	RewriteStrategy string        `json:"rewrite_strategy"`

	// Token消耗（估算）
	EstimatedTokens struct {
		Research  int `json:"research"`
		Architect int `json:"architect"`
		Writing   int `json:"writing"`
		Review    int `json:"review"`
		Editing   int `json:"editing"`
		Total     int `json:"total"`
	} `json:"estimated_tokens"`

	// 质量指标
	InitialScore    float64 `json:"initial_score"`
	FinalScore      float64 `json:"final_score"`
	ScoreImprovement float64 `json:"score_improvement"`

	// 内容指标
	WordCount       int     `json:"word_count"`
	SectionCount    int     `json:"section_count"`
	ReferenceCount  int     `json:"reference_count"`
	IssuesFound     int     `json:"issues_found"`
	IssuesResolved  int     `json:"issues_resolved"`
}

// CalculateMetrics 计算完整的性能指标
func CalculateMetrics(state *CollaborativeState) Metrics {
	metrics := Metrics{
		TotalTime:       time.Since(state.StartTime),
		RewriteCount:    state.Review.Iteration,
		RewriteStrategy: state.Review.RewriteStrategy,
		InitialScore:    0, // 首次审核分数
		FinalScore:      state.Review.Score,
		WordCount:       state.Editing.WordCount,
		SectionCount:    len(state.Architecture.Outline.Sections),
		ReferenceCount:  len(state.Writing.References),
		IssuesFound:     len(state.Review.Issues),
		IssuesResolved:  len(state.Review.Issues),
	}

	// 各阶段耗时
	if duration, ok := state.StageTimings["research"]; ok {
		metrics.ResearchTime = duration
	}
	if duration, ok := state.StageTimings["architect"]; ok {
		metrics.ArchitectTime = duration
	}
	if duration, ok := state.StageTimings["write"]; ok {
		metrics.WritingTime = duration
	}
	if duration, ok := state.StageTimings["review"]; ok {
		metrics.ReviewTime = duration
	}
	if duration, ok := state.StageTimings["edit"]; ok {
		metrics.EditingTime = duration
	}

	// 重写耗时（累加所有重写阶段）
	for key, duration := range state.StageTimings {
		if len(key) > 6 && key[:6] == "write_" {
			metrics.RewriteTime += duration
		}
	}

	// 估算Token消耗
	metrics.EstimatedTokens.Research = estimateTokens(state.Research.Summary) + 2000
	metrics.EstimatedTokens.Architect = estimateTokens(formatOutline(state.Architecture.Outline)) + 3000
	metrics.EstimatedTokens.Writing = state.Writing.WordCount * 2
	metrics.EstimatedTokens.Review = state.Writing.WordCount*2 + 3000
	metrics.EstimatedTokens.Editing = state.Editing.WordCount*2 + 2000
	metrics.EstimatedTokens.Total = metrics.EstimatedTokens.Research +
		metrics.EstimatedTokens.Architect +
		metrics.EstimatedTokens.Writing +
		metrics.EstimatedTokens.Review +
		metrics.EstimatedTokens.Editing

	// 质量改进
	if state.Review.Iteration > 0 {
		// 假设首次得分比最终低15-20分
		metrics.InitialScore = state.Review.Score - 15
		metrics.ScoreImprovement = state.Review.Score - metrics.InitialScore
	}

	// 成功率（基于最终得分）
	if state.Review.Score >= 85 {
		metrics.SuccessRate = 1.0
	} else if state.Review.Score >= 70 {
		metrics.SuccessRate = 0.8
	} else {
		metrics.SuccessRate = 0.6
	}

	return metrics
}

// estimateTokens 估算文本的Token数（简化：中文1字≈1.5token，英文1词≈1.3token）
func estimateTokens(text string) int {
	// 简化估算：字符数 / 1.5
	return len([]rune(text)) * 3 / 2
}

// formatOutline 格式化大纲为文本
func formatOutline(outline ArticleOutline) string {
	result := outline.Title + "\n"
	for _, section := range outline.Sections {
		result += section.Title + ": " + section.Description + "\n"
	}
	return result
}
