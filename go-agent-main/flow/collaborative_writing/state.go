package collaborative_writing

import (
	"time"
)

// CollaborativeState 多Agent协作的共享状态
type CollaborativeState struct {
	// ===== 输入 =====
	Topic        string
	Requirements string
	AuthorID     uint
	RequestTime  time.Time

	// ===== 研究阶段 =====
	Research ResearchOutput

	// ===== 架构阶段 =====
	Architecture ArchitectureOutput

	// ===== 写作阶段 =====
	Writing WritingOutput

	// Phase 2: 保存之前的写作结果，用于增量重写
	PreviousWriting *WritingOutput

	// ===== 审核阶段 =====
	Review ReviewOutput

	// ===== 编辑阶段 =====
	Editing EditingOutput

	// ===== 元数据 =====
	CurrentStage string
	StartTime    time.Time
	StageTimings map[string]time.Duration
}

// ResearchOutput 研究员Agent的输出
type ResearchOutput struct {
	Articles    []ArticleReference
	Summary     string   // 压缩后的摘要（给后续Agent）
	KeyFindings []string // 关键发现
	TotalWords  int
	Completed   bool
}

// ArticleReference 参考文章
type ArticleReference struct {
	ID      uint
	Title   string
	Content string
	Tags    []string
	URL     string
}

// ArchitectureOutput 架构师Agent的输出
type ArchitectureOutput struct {
	Outline     ArticleOutline
	Rationale   string // 为什么这样设计
	TargetWords int
	Completed   bool
}

// ArticleOutline 文章大纲
type ArticleOutline struct {
	Title       string
	Subtitle    string
	Sections    []OutlineSection
	TotalWords  int
	Difficulty  string // "beginner", "intermediate", "advanced"
	Tone        string // "formal", "casual", "technical"
}

// OutlineSection 大纲章节
type OutlineSection struct {
	Order       int
	Title       string
	Description string
	KeyPoints   []string
	TargetWords int
}

// WritingOutput 作家Agent的输出
type WritingOutput struct {
	Content    string
	WordCount  int
	References []Reference
	Completed  bool

	// Phase 2: 章节独立存储，支持增量重写
	SectionContents map[int]string // 章节序号 → 内容
}

// Reference 引用
type Reference struct {
	Number  int
	Title   string
	URL     string
	Context string // 在哪里引用的
}

// ReviewOutput 审核Agent的输出
type ReviewOutput struct {
	Approved    bool
	Score       float64 // 0-100分
	Issues      []ReviewIssue
	Suggestions []string
	Iteration   int // 重写次数
	Completed   bool

	// Phase 2: 增量重写支持
	ProblematicSections []int  // 有问题的章节序号
	RewriteStrategy     string // "full" 或 "incremental"
}

// ReviewIssue 审核问题
type ReviewIssue struct {
	Type        string // "quality", "risk", "structure", "grammar"
	Severity    string // "low", "medium", "high", "critical"
	Description string
	Position    string // 问题位置
	Suggestion  string // 修改建议

	// Phase 2: 问题所在章节
	SectionOrder int // 问题所在章节序号（0表示全局问题）
}

// EditingOutput 编辑Agent的输出
type EditingOutput struct {
	FinalContent string
	Changes      []EditChange
	WordCount    int
	Completed    bool
}

// EditChange 编辑修改记录
type EditChange struct {
	Type        string // "grammar", "style", "structure", "enhancement"
	Description string
	Before      string
	After       string
}

// WritingRequest 写作请求
type WritingRequest struct {
	Topic        string `json:"topic" binding:"required"`
	Requirements string `json:"requirements,omitempty"`
	AuthorID     uint   `json:"author_id" binding:"required"`
	TargetWords  int    `json:"target_words,omitempty"` // 目标字数
	Difficulty   string `json:"difficulty,omitempty"`   // 难度级别
	Tone         string `json:"tone,omitempty"`         // 语气风格
}

// WritingResult 写作结果
type WritingResult struct {
	Success      bool              `json:"success"`
	ArticleID    uint              `json:"article_id,omitempty"`
	Content      string            `json:"content,omitempty"`
	WordCount    int               `json:"word_count"`
	ReviewScore  float64           `json:"review_score"`
	Iterations   int               `json:"iterations"` // 经历了几次重写
	TotalTime    time.Duration     `json:"total_time"`
	StageTimings map[string]string `json:"stage_timings"`
	Error        string            `json:"error,omitempty"`

	// Phase 2: 重写详情
	RewriteStrategy     string   `json:"rewrite_strategy,omitempty"`      // "full" 或 "incremental"
	ProblematicSections []int    `json:"problematic_sections,omitempty"`  // 有问题的章节
	IssuesResolved      int      `json:"issues_resolved,omitempty"`       // 解决的问题数

	// Phase 3: 详细监控指标
	Metrics *Metrics `json:"metrics,omitempty"` // 详细性能指标
}
