package collaborative_writing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ReviewerAgent 审核Agent - 负责检查文章质量和风险
type ReviewerAgent struct {
	model model.BaseChatModel
}

// NewReviewerAgent 创建审核Agent
func NewReviewerAgent(chatModel model.BaseChatModel) *ReviewerAgent {
	return &ReviewerAgent{
		model: chatModel,
	}
}

// Review 审核文章
func (a *ReviewerAgent) Review(
	ctx context.Context,
	topic string,
	outline ArticleOutline,
	writing WritingOutput,
	currentIteration int,
) (ReviewOutput, error) {
	log.Printf("[ReviewerAgent] 开始审核文章（第%d次）", currentIteration+1)

	// 构建审核Prompt
	prompt := a.buildReviewPrompt(topic, outline, writing, currentIteration)

	// 调用LLM审核
	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是严格的技术文章审核专家，从质量、风险、结构等多个维度评估文章。只输出JSON格式。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return ReviewOutput{}, fmt.Errorf("review failed: %w", err)
	}

	// 解析JSON
	var result struct {
		Approved    bool          `json:"approved"`
		Score       float64       `json:"score"`
		Issues      []ReviewIssue `json:"issues"`
		Suggestions []string      `json:"suggestions"`
	}

	content := extractJSONFromResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ReviewOutput{}, fmt.Errorf("parse review JSON: %w (content: %s)", err, content)
	}

	// 评分低于70分视为不通过
	if result.Score < 70 && result.Approved {
		result.Approved = false
	}

	// Phase 2: 分析问题章节并决定重写策略
	problematicSections := a.identifyProblematicSections(result.Issues, outline)
	rewriteStrategy := a.determineRewriteStrategy(result.Issues, currentIteration)

	if result.Approved {
		log.Printf("[ReviewerAgent] ✅ 文章通过审核，得分: %.1f", result.Score)
	} else {
		log.Printf("[ReviewerAgent] ❌ 文章未通过审核，得分: %.1f，问题数: %d", result.Score, len(result.Issues))
		log.Printf("[ReviewerAgent] 问题章节: %v，重写策略: %s", problematicSections, rewriteStrategy)
	}

	return ReviewOutput{
		Approved:            result.Approved,
		Score:               result.Score,
		Issues:              result.Issues,
		Suggestions:         result.Suggestions,
		Iteration:           currentIteration,
		Completed:           true,
		ProblematicSections: problematicSections,
		RewriteStrategy:     rewriteStrategy,
	}, nil
}

// buildReviewPrompt 构建审核Prompt
func (a *ReviewerAgent) buildReviewPrompt(
	topic string,
	outline ArticleOutline,
	writing WritingOutput,
	iteration int,
) string {
	iterationNote := ""
	if iteration > 0 {
		iterationNote = fmt.Sprintf("\n【注意】这是第%d次重写后的版本，请重点检查是否改进了之前的问题。\n", iteration+1)
	}

	return fmt.Sprintf(`审核以下技术文章，从多个维度评估质量。

主题：%s
预期标题：%s
实际字数：%d字
目标字数：%d字
%s
文章内容：
%s

审核维度：
1. **内容质量** (30分)
   - 技术准确性
   - 内容深度和广度
   - 逻辑清晰性
   - 是否偏离主题

2. **结构完整性** (20分)
   - 是否覆盖大纲所有要点
   - 章节衔接是否流畅
   - 是否有引言和总结

3. **可读性** (20分)
   - 语言流畅度
   - 是否易于理解
   - 格式规范性（Markdown）

4. **实用性** (20分)
   - 是否有实际案例
   - 是否有代码示例（如适用）
   - 是否有可操作的建议

5. **风险检查** (10分)
   - 是否有敏感内容
   - 是否有明显错误
   - 是否有侵权风险

输出JSON格式：
{
  "approved": true/false,
  "score": 85.5,
  "issues": [
    {
      "type": "quality/risk/structure/grammar",
      "severity": "low/medium/high/critical",
      "description": "问题描述",
      "position": "问题位置（章节名）",
      "suggestion": "改进建议"
    }
  ],
  "suggestions": [
    "整体改进建议1",
    "整体改进建议2"
  ]
}

审核标准：
- 得分≥85分且无critical问题：通过
- 得分70-84分：视issues严重程度决定
- 得分<70分：不通过，需要重写

只输出JSON，不要其他内容。`, topic, outline.Title, writing.WordCount, outline.TotalWords, iterationNote, writing.Content)
}

// identifyProblematicSections Phase 2: 识别有问题的章节
func (a *ReviewerAgent) identifyProblematicSections(issues []ReviewIssue, outline ArticleOutline) []int {
	sectionMap := make(map[int]bool)

	for _, issue := range issues {
		// 尝试从position字段提取章节序号
		if issue.Position != "" {
			for _, section := range outline.Sections {
				if strings.Contains(issue.Position, section.Title) {
					sectionMap[section.Order] = true
					break
				}
			}
		}

		// 如果issue已经标记了SectionOrder
		if issue.SectionOrder > 0 {
			sectionMap[issue.SectionOrder] = true
		}
	}

	// 转换为有序列表
	var sections []int
	for sectionOrder := range sectionMap {
		sections = append(sections, sectionOrder)
	}

	// 排序
	sort.Ints(sections)

	return sections
}

// determineRewriteStrategy Phase 2: 决定重写策略
func (a *ReviewerAgent) determineRewriteStrategy(issues []ReviewIssue, iteration int) string {
	// 第一次重写：如果问题集中在少数章节，使用增量重写
	if iteration == 0 {
		criticalCount := 0
		for _, issue := range issues {
			if issue.Severity == "critical" || issue.Severity == "high" {
				criticalCount++
			}
		}

		// 如果严重问题少于3个，尝试增量重写
		if criticalCount < 3 {
			return "incremental"
		}
	}

	// 其他情况：全部重写
	return "full"
}
