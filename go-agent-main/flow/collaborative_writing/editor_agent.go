package collaborative_writing

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EditorAgent 编辑Agent - 负责润色优化文章
type EditorAgent struct {
	model model.BaseChatModel
}

// NewEditorAgent 创建编辑Agent
func NewEditorAgent(chatModel model.BaseChatModel) *EditorAgent {
	return &EditorAgent{
		model: chatModel,
	}
}

// Polish 润色文章
func (a *EditorAgent) Polish(
	ctx context.Context,
	writing WritingOutput,
	review ReviewOutput,
) (EditingOutput, error) {
	log.Printf("[EditorAgent] 开始润色文章，原文%d字", writing.WordCount)

	// 如果审核得分很高，只做轻微润色
	if review.Score >= 90 {
		log.Printf("[EditorAgent] 文章质量优秀(%.1f分)，仅做轻微润色", review.Score)
		return a.lightPolish(ctx, writing)
	}

	// 否则进行深度润色
	log.Printf("[EditorAgent] 进行深度润色优化")
	return a.deepPolish(ctx, writing, review)
}

// lightPolish 轻微润色
func (a *EditorAgent) lightPolish(ctx context.Context, writing WritingOutput) (EditingOutput, error) {
	prompt := fmt.Sprintf(`对以下技术文章进行轻微润色，主要修正：
1. 明显的语法错误
2. 标点符号规范
3. Markdown格式优化
4. 措辞优化（保持原意）

原文：
%s

直接输出润色后的完整文章，不要其他说明。`, writing.Content)

	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是专业技术文章编辑，擅长优化文字和格式。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return EditingOutput{}, err
	}

	polishedContent := response.Content
	wordCount := len([]rune(polishedContent))

	changes := []EditChange{
		{
			Type:        "light_polish",
			Description: "修正语法、标点、格式",
		},
	}

	log.Printf("[EditorAgent] 轻微润色完成: %d字", wordCount)

	return EditingOutput{
		FinalContent: polishedContent,
		Changes:      changes,
		WordCount:    wordCount,
		Completed:    true,
	}, nil
}

// deepPolish 深度润色
func (a *EditorAgent) deepPolish(ctx context.Context, writing WritingOutput, review ReviewOutput) (EditingOutput, error) {
	// 根据审核意见构建润色重点
	focusAreas := a.buildFocusAreas(review)

	prompt := fmt.Sprintf(`对以下技术文章进行深度润色优化。

审核得分：%.1f分
重点优化方向：
%s

原文：
%s

润色要求：
1. 保持原文核心内容和结构
2. 优化语言表达，提升可读性
3. 修正所有语法和格式问题
4. 增强逻辑连贯性
5. 适当优化段落和章节过渡
6. 保持Markdown格式

直接输出润色后的完整文章，不要其他说明。`, review.Score, focusAreas, writing.Content)

	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是资深技术文章编辑，擅长深度优化和润色。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return EditingOutput{}, err
	}

	polishedContent := response.Content
	wordCount := len([]rune(polishedContent))

	// 记录修改类型
	changes := a.extractChanges(review)

	log.Printf("[EditorAgent] 深度润色完成: %d字 → %d字", writing.WordCount, wordCount)

	return EditingOutput{
		FinalContent: polishedContent,
		Changes:      changes,
		WordCount:    wordCount,
		Completed:    true,
	}, nil
}

// buildFocusAreas 构建润色重点
func (a *EditorAgent) buildFocusAreas(review ReviewOutput) string {
	var areas []string

	// 根据issues统计问题类型
	issueTypes := make(map[string]int)
	for _, issue := range review.Issues {
		issueTypes[issue.Type]++
	}

	for issueType, count := range issueTypes {
		typeDesc := map[string]string{
			"quality":   "内容质量",
			"structure": "结构完整性",
			"grammar":   "语法和措辞",
			"risk":      "风险内容",
		}[issueType]

		if typeDesc != "" {
			areas = append(areas, fmt.Sprintf("- %s (%d处问题)", typeDesc, count))
		}
	}

	// 添加审核建议
	for i, suggestion := range review.Suggestions {
		if i < 3 { // 最多3条
			areas = append(areas, fmt.Sprintf("- %s", suggestion))
		}
	}

	if len(areas) == 0 {
		return "- 全面提升文章质量和可读性"
	}

	return strings.Join(areas, "\n")
}

// extractChanges 提取修改类型
func (a *EditorAgent) extractChanges(review ReviewOutput) []EditChange {
	changes := []EditChange{}

	// 根据问题类型生成修改记录
	issueTypes := make(map[string]bool)
	for _, issue := range review.Issues {
		issueTypes[issue.Type] = true
	}

	if issueTypes["grammar"] {
		changes = append(changes, EditChange{
			Type:        "grammar",
			Description: "修正语法和标点错误",
		})
	}

	if issueTypes["structure"] {
		changes = append(changes, EditChange{
			Type:        "structure",
			Description: "优化文章结构和逻辑",
		})
	}

	if issueTypes["quality"] {
		changes = append(changes, EditChange{
			Type:        "enhancement",
			Description: "提升内容质量和深度",
		})
	}

	changes = append(changes, EditChange{
		Type:        "style",
		Description: "优化语言表达和可读性",
	})

	return changes
}
