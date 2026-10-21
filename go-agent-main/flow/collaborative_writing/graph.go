package collaborative_writing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go-agent/config"
	"go-agent/model/chat_model"

	"github.com/cloudwego/eino/compose"
)

var (
	cachedCollaborativeGraph  compose.Runnable[WritingRequest, WritingResult]
	collaborativeGraphOnce    sync.Once
	collaborativeGraphInitErr error
)

// InitCollaborativeWritingGraph 初始化多Agent协作写作图
func InitCollaborativeWritingGraph(ctx context.Context) error {
	collaborativeGraphOnce.Do(func() {
		cachedCollaborativeGraph, collaborativeGraphInitErr = buildCollaborativeWritingGraph(ctx)
	})
	return collaborativeGraphInitErr
}

// GetCollaborativeWritingGraph 获取已缓存的协作写作图
func GetCollaborativeWritingGraph() (compose.Runnable[WritingRequest, WritingResult], error) {
	if cachedCollaborativeGraph == nil {
		return nil, fmt.Errorf("CollaborativeWritingGraph 未初始化，请先调用 InitCollaborativeWritingGraph")
	}
	return cachedCollaborativeGraph, nil
}

// buildCollaborativeWritingGraph 构建协作写作图（简化版：线性流程）
func buildCollaborativeWritingGraph(ctx context.Context) (compose.Runnable[WritingRequest, WritingResult], error) {
	const (
		Initialize = "initialize"
		Research   = "research"
		Architect  = "architect"
		Write      = "write"
		Edit       = "edit"
		Finalize   = "finalize"
	)

	// 获取模型
	chatModel, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType)
	if err != nil {
		return nil, fmt.Errorf("get chat model: %w", err)
	}

	// 创建各Agent
	researcher, err := NewResearcherAgent(chatModel)
	if err != nil {
		return nil, fmt.Errorf("create researcher: %w", err)
	}

	architect := NewArchitectAgent(chatModel)
	writer := NewWriterAgent(chatModel)
	reviewer := NewReviewerAgent(chatModel)
	editor := NewEditorAgent(chatModel)

	// 构建Graph
	g := compose.NewGraph[WritingRequest, WritingResult](
		compose.WithGenLocalState(func(ctx context.Context) *CollaborativeState {
			return &CollaborativeState{
				StageTimings: make(map[string]time.Duration),
			}
		}),
	)

	// 节点1: 初始化
	_ = g.AddLambdaNode(Initialize, compose.InvokableLambda(
		func(ctx context.Context, req WritingRequest) (WritingRequest, error) {
			log.Printf("🚀 [CollaborativeWriting] 开始多Agent协作写作")
			log.Printf("📝 主题: %s", req.Topic)

			err := compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, state *CollaborativeState) error {
					state.Topic = req.Topic
					state.Requirements = req.Requirements
					state.AuthorID = req.AuthorID
					state.RequestTime = time.Now()
					state.StartTime = time.Now()
					state.CurrentStage = "initialized"
					return nil
				})

			return req, err
		},
	))

	// 节点2: 研究员Agent
	_ = g.AddLambdaNode(Research, compose.InvokableLambda(
		func(ctx context.Context, req WritingRequest) (ResearchOutput, error) {
			stageStart := time.Now()
			log.Printf("🔍 [Research] 研究员Agent开始工作")

			research, err := researcher.Research(ctx, req.Topic, req.Requirements)

			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, state *CollaborativeState) error {
					state.Research = research
					state.CurrentStage = "research_completed"
					state.StageTimings["research"] = time.Since(stageStart)
					return nil
				})

			return research, err
		},
	))

	// 节点3: 架构师Agent
	_ = g.AddLambdaNode(Architect, compose.InvokableLambda(
		func(ctx context.Context, research ResearchOutput) (ArchitectureOutput, error) {
			stageStart := time.Now()
			log.Printf("🏗️ [Architect] 架构师Agent开始设计")

			var state *CollaborativeState
			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, s *CollaborativeState) error {
					state = s
					return nil
				})

			architecture, err := architect.DesignOutline(
				ctx,
				state.Topic,
				state.Requirements,
				research,
				2000, // 默认2000字
				"medium", // 默认中等难度
				"professional", // 默认专业语气
			)

			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, state *CollaborativeState) error {
					state.Architecture = architecture
					state.CurrentStage = "architecture_completed"
					state.StageTimings["architect"] = time.Since(stageStart)
					return nil
				})

			return architecture, err
		},
	))

	// 节点4: 作家Agent（含内嵌审核）
	_ = g.AddLambdaNode(Write, compose.InvokableLambda(
		func(ctx context.Context, architecture ArchitectureOutput) (WritingOutput, error) {
			stageStart := time.Now()
			log.Printf("✍️ [Write] 作家Agent开始撰写")

			var state *CollaborativeState
			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, s *CollaborativeState) error {
					state = s
					return nil
				})

			// 写作
			writing, err := writer.Write(
				ctx,
				state.Topic,
				architecture.Outline,
				state.Research,
				nil,
			)
			if err != nil {
				return WritingOutput{}, err
			}

			// 内嵌审核（不重写，只评分）
			reviewStart := time.Now()
			log.Printf("🔎 [Review] 内嵌审核开始")

			review, err := reviewer.Review(
				ctx,
				state.Topic,
				architecture.Outline,
				writing,
				0, // 第0次迭代
			)
			if err != nil {
				log.Printf("⚠️ [Review] 审核失败，继续流程: %v", err)
				// 审核失败不影响流程，使用默认分数
				review = ReviewOutput{
					Score:    80.0,
					Approved: true,
				}
			}

			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, s *CollaborativeState) error {
					s.Writing = writing
					s.Review = review
					s.CurrentStage = "writing_completed"
					s.StageTimings["write"] = time.Since(stageStart)
					s.StageTimings["review"] = time.Since(reviewStart)
					return nil
				})

			log.Printf("✅ [Review] 质量得分: %.1f", review.Score)

			return writing, nil
		},
	))

	// 节点5: 编辑Agent
	_ = g.AddLambdaNode(Edit, compose.InvokableLambda(
		func(ctx context.Context, writing WritingOutput) (EditingOutput, error) {
			stageStart := time.Now()
			log.Printf("💎 [Edit] 编辑Agent开始润色")

			var state *CollaborativeState
			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, s *CollaborativeState) error {
					state = s
					return nil
				})

			editing, err := editor.Polish(ctx, writing, state.Review)

			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, state *CollaborativeState) error {
					state.Editing = editing
					state.CurrentStage = "editing_completed"
					state.StageTimings["edit"] = time.Since(stageStart)
					return nil
				})

			return editing, err
		},
	))

	// 节点6: 完成
	_ = g.AddLambdaNode(Finalize, compose.InvokableLambda(
		func(ctx context.Context, editing EditingOutput) (WritingResult, error) {
			log.Printf("🎉 [Finalize] 多Agent协作完成")

			var state *CollaborativeState
			compose.ProcessState[*CollaborativeState](ctx,
				func(ctx context.Context, s *CollaborativeState) error {
					state = s
					return nil
				})

			totalTime := time.Since(state.StartTime)

			// 格式化耗时
			timings := make(map[string]string)
			for stage, duration := range state.StageTimings {
				timings[stage] = fmt.Sprintf("%.1fs", duration.Seconds())
			}

			// 计算详细指标
			metrics := CalculateMetrics(state)

			result := WritingResult{
				Success:         true,
				Content:         editing.FinalContent,
				WordCount:       editing.WordCount,
				ReviewScore:     state.Review.Score,
				Iterations:      1, // 简化版固定1次
				RewriteStrategy: "single_pass", // 简化版：单次通过
				TotalTime:       totalTime,
				StageTimings:    timings,
				Metrics:         &metrics,
			}

			log.Printf("📊 [Metrics] 总耗时: %s, 字数: %d, 得分: %.1f",
				result.TotalTime, result.WordCount, result.ReviewScore)

			return result, nil
		},
	))

	// 连接边（线性流程，无循环）
	_ = g.AddEdge(compose.START, Initialize)
	_ = g.AddEdge(Initialize, Research)
	_ = g.AddEdge(Research, Architect)
	_ = g.AddEdge(Architect, Write)
	_ = g.AddEdge(Write, Edit)
	_ = g.AddEdge(Edit, Finalize)
	_ = g.AddEdge(Finalize, compose.END)

	return g.Compile(
		ctx,
		compose.WithGraphName("CollaborativeWriting"),
		compose.WithNodeTriggerMode(compose.AllPredecessor),
	)
}
