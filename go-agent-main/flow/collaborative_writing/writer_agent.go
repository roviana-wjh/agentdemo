package collaborative_writing

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// WriterAgent 作家Agent - 负责根据大纲撰写文章
type WriterAgent struct {
	model model.BaseChatModel
}

// NewWriterAgent 创建作家Agent
func NewWriterAgent(chatModel model.BaseChatModel) *WriterAgent {
	return &WriterAgent{
		model: chatModel,
	}
}

// Write 撰写文章
func (a *WriterAgent) Write(
	ctx context.Context,
	topic string,
	outline ArticleOutline,
	research ResearchOutput,
	reviewFeedback *ReviewOutput, // 如果是重写，包含审核反馈
) (WritingOutput, error) {
	if reviewFeedback != nil && reviewFeedback.Iteration > 0 {
		log.Printf("[WriterAgent] 第%d次重写，根据审核反馈改进", reviewFeedback.Iteration)

		// Phase 2: 增量重写
		if reviewFeedback.RewriteStrategy == "incremental" && len(reviewFeedback.ProblematicSections) > 0 {
			log.Printf("[WriterAgent] 使用增量重写策略，仅重写章节: %v", reviewFeedback.ProblematicSections)
			return a.incrementalRewrite(ctx, topic, outline, research, reviewFeedback)
		}

		log.Printf("[WriterAgent] 使用全文重写策略")
	} else {
		log.Printf("[WriterAgent] 开始撰写文章: %s", outline.Title)
	}

	// 逐章节撰写
	var sections []string
	var references []Reference
	refCounter := 1
	sectionContents := make(map[int]string)

	for _, section := range outline.Sections {
		log.Printf("[WriterAgent] 撰写章节 %d: %s", section.Order, section.Title)

		sectionContent, sectionRefs, err := a.writeSection(
			ctx,
			topic,
			section,
			outline,
			research,
			reviewFeedback,
		)
		if err != nil {
			return WritingOutput{}, fmt.Errorf("write section %d: %w", section.Order, err)
		}

		sections = append(sections, sectionContent)
		sectionContents[section.Order] = sectionContent

		// 收集引用
		for _, ref := range sectionRefs {
			ref.Number = refCounter
			references = append(references, ref)
			refCounter++
		}
	}

	// 组装完整文章
	fullContent := a.assembleArticle(outline, sections, references)

	wordCount := len([]rune(fullContent))
	log.Printf("[WriterAgent] 文章撰写完成: %d字，%d个引用", wordCount, len(references))

	return WritingOutput{
		Content:         fullContent,
		WordCount:       wordCount,
		References:      references,
		Completed:       true,
		SectionContents: sectionContents,
	}, nil
}

// writeSection 撰写单个章节
func (a *WriterAgent) writeSection(
	ctx context.Context,
	topic string,
	section OutlineSection,
	outline ArticleOutline,
	research ResearchOutput,
	reviewFeedback *ReviewOutput,
) (string, []Reference, error) {
	// 构建章节写作Prompt
	prompt := a.buildSectionPrompt(topic, section, outline, research, reviewFeedback)

	// 调用LLM生成
	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(a.getWriterPersona(outline.Difficulty, outline.Tone)),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return "", nil, err
	}

	content := response.Content

	// 提取引用（简化处理）
	var references []Reference
	if research.Completed && len(research.Articles) > 0 {
		// 检查是否引用了研究文章
		for _, article := range research.Articles {
			if strings.Contains(content, article.Title) {
				references = append(references, Reference{
					Title:   article.Title,
					URL:     article.URL,
					Context: section.Title,
				})
			}
		}
	}

	return content, references, nil
}

// buildSectionPrompt 构建章节写作Prompt
func (a *WriterAgent) buildSectionPrompt(
	topic string,
	section OutlineSection,
	outline ArticleOutline,
	research ResearchOutput,
	reviewFeedback *ReviewOutput,
) string {
	// 基础信息
	prompt := fmt.Sprintf(`撰写技术文章的章节内容。

文章主题：%s
文章标题：%s
当前章节：第%d章 - %s

章节要求：
- 描述：%s
- 关键要点：%v
- 目标字数：%d字左右

`, topic, outline.Title, section.Order, section.Title, section.Description, section.KeyPoints, section.TargetWords)

	// 添加研究背景
	if research.Completed && research.Summary != "" {
		prompt += fmt.Sprintf(`
参考研究摘要：
%s

`, research.Summary)
	}

	// 添加审核反馈（如果是重写）
	if reviewFeedback != nil && len(reviewFeedback.Issues) > 0 {
		prompt += "\n【重要】请根据以下审核反馈改进：\n"
		for i, issue := range reviewFeedback.Issues {
			prompt += fmt.Sprintf("%d. [%s] %s\n   建议：%s\n", i+1, issue.Severity, issue.Description, issue.Suggestion)
		}
		prompt += "\n"
	}

	// 写作要求
	prompt += `
写作要求：
1. 内容充实，逻辑清晰
2. 技术准确，有深度
3. 包含具体示例或代码（如适用）
4. 语言流畅，易于理解
5. 使用Markdown格式
6. 不要输出章节标题（会统一添加）

直接输出章节正文内容，不要其他说明。`

	return prompt
}

// getWriterPersona 获取作家人格设定
func (a *WriterAgent) getWriterPersona(difficulty string, tone string) string {
	difficultyPersona := map[string]string{
		"beginner":     "你是技术科普作家，擅长用通俗易懂的语言解释复杂概念，适合初学者阅读。",
		"intermediate": "你是资深技术博主，有丰富的实战经验，擅长深入浅出讲解技术实践。",
		"advanced":     "你是技术专家，精通底层原理和架构设计，擅长深度技术剖析。",
	}[difficulty]

	tonePersona := map[string]string{
		"formal":    "写作风格正式、严谨，使用学术化表达。",
		"casual":    "写作风格轻松、亲切，使用口语化表达，像和朋友聊天。",
		"technical": "写作风格专业、准确，注重技术细节和精确表述。",
	}[tone]

	return fmt.Sprintf("%s %s 使用Markdown格式输出。", difficultyPersona, tonePersona)
}

// assembleArticle 组装完整文章
func (a *WriterAgent) assembleArticle(
	outline ArticleOutline,
	sections []string,
	references []Reference,
) string {
	var builder strings.Builder

	// 标题
	builder.WriteString(fmt.Sprintf("# %s\n\n", outline.Title))

	// 副标题
	if outline.Subtitle != "" {
		builder.WriteString(fmt.Sprintf("*%s*\n\n", outline.Subtitle))
	}

	// 分隔线
	builder.WriteString("---\n\n")

	// 逐章节添加
	for i, section := range outline.Sections {
		// 章节标题
		builder.WriteString(fmt.Sprintf("## %d. %s\n\n", section.Order, section.Title))

		// 章节内容
		builder.WriteString(sections[i])
		builder.WriteString("\n\n")
	}

	// 引用列表
	if len(references) > 0 {
		builder.WriteString("---\n\n")
		builder.WriteString("## 参考资料\n\n")
		for _, ref := range references {
			builder.WriteString(fmt.Sprintf("[%d] [%s](%s)\n", ref.Number, ref.Title, ref.URL))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// incrementalRewrite Phase 2: 增量重写（只改有问题的章节）
func (a *WriterAgent) incrementalRewrite(
	ctx context.Context,
	topic string,
	outline ArticleOutline,
	research ResearchOutput,
	reviewFeedback *ReviewOutput,
) (WritingOutput, error) {
	log.Printf("[WriterAgent] 执行增量重写，问题章节: %v", reviewFeedback.ProblematicSections)

	// 从State获取之前的章节内容（需要从Graph State传递）
	// 这里简化处理：重写问题章节，其他章节保持原样

	var sections []string
	var references []Reference
	refCounter := 1
	sectionContents := make(map[int]string)

	problematicSet := make(map[int]bool)
	for _, order := range reviewFeedback.ProblematicSections {
		problematicSet[order] = true
	}

	for _, section := range outline.Sections {
		var sectionContent string
		var sectionRefs []Reference
		var err error

		if problematicSet[section.Order] {
			// 重写有问题的章节
			log.Printf("[WriterAgent] 🔄 重写章节 %d: %s", section.Order, section.Title)

			// 过滤出该章节的问题
			sectionIssues := a.filterIssuesForSection(reviewFeedback.Issues, section.Order)

			sectionContent, sectionRefs, err = a.rewriteSection(
				ctx,
				topic,
				section,
				outline,
				research,
				sectionIssues,
			)
			if err != nil {
				return WritingOutput{}, fmt.Errorf("rewrite section %d: %w", section.Order, err)
			}
		} else {
			// 保持原章节（如果有的话，这里简化为重新生成）
			log.Printf("[WriterAgent] ✓ 保持章节 %d: %s", section.Order, section.Title)
			sectionContent, sectionRefs, err = a.writeSection(
				ctx,
				topic,
				section,
				outline,
				research,
				nil, // 不传反馈
			)
			if err != nil {
				return WritingOutput{}, fmt.Errorf("keep section %d: %w", section.Order, err)
			}
		}

		sections = append(sections, sectionContent)
		sectionContents[section.Order] = sectionContent

		// 收集引用
		for _, ref := range sectionRefs {
			ref.Number = refCounter
			references = append(references, ref)
			refCounter++
		}
	}

	// 组装完整文章
	fullContent := a.assembleArticle(outline, sections, references)
	wordCount := len([]rune(fullContent))

	log.Printf("[WriterAgent] 增量重写完成: %d字", wordCount)

	return WritingOutput{
		Content:         fullContent,
		WordCount:       wordCount,
		References:      references,
		Completed:       true,
		SectionContents: sectionContents,
	}, nil
}

// filterIssuesForSection 过滤出特定章节的问题
func (a *WriterAgent) filterIssuesForSection(issues []ReviewIssue, sectionOrder int) []ReviewIssue {
	var filtered []ReviewIssue
	for _, issue := range issues {
		if issue.SectionOrder == sectionOrder {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// rewriteSection 重写单个章节（针对性改进）
func (a *WriterAgent) rewriteSection(
	ctx context.Context,
	topic string,
	section OutlineSection,
	outline ArticleOutline,
	research ResearchOutput,
	sectionIssues []ReviewIssue,
) (string, []Reference, error) {
	// 构建针对性重写Prompt
	prompt := a.buildRewriteSectionPrompt(topic, section, outline, research, sectionIssues)

	// 调用LLM生成
	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(a.getWriterPersona(outline.Difficulty, outline.Tone)),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return "", nil, err
	}

	content := response.Content

	// 提取引用
	var references []Reference
	if research.Completed && len(research.Articles) > 0 {
		for _, article := range research.Articles {
			if strings.Contains(content, article.Title) {
				references = append(references, Reference{
					Title:   article.Title,
					URL:     article.URL,
					Context: section.Title,
				})
			}
		}
	}

	return content, references, nil
}

// buildRewriteSectionPrompt 构建针对性重写Prompt
func (a *WriterAgent) buildRewriteSectionPrompt(
	topic string,
	section OutlineSection,
	outline ArticleOutline,
	research ResearchOutput,
	sectionIssues []ReviewIssue,
) string {
	prompt := fmt.Sprintf(`重写技术文章的章节，针对性解决审核发现的问题。

文章主题：%s
文章标题：%s
当前章节：第%d章 - %s

章节要求：
- 描述：%s
- 关键要点：%v
- 目标字数：%d字左右

`, topic, outline.Title, section.Order, section.Title, section.Description, section.KeyPoints, section.TargetWords)

	// 添加问题和改进建议
	if len(sectionIssues) > 0 {
		prompt += "\n【本章节存在以下问题，请重点改进】：\n"
		for i, issue := range sectionIssues {
			prompt += fmt.Sprintf("%d. [%s级] %s\n", i+1, issue.Severity, issue.Description)
			if issue.Suggestion != "" {
				prompt += fmt.Sprintf("   改进建议：%s\n", issue.Suggestion)
			}
		}
		prompt += "\n"
	}

	// 添加研究背景
	if research.Completed && research.Summary != "" {
		prompt += fmt.Sprintf(`
参考研究摘要：
%s

`, research.Summary)
	}

	prompt += `
重写要求：
1. 针对性解决上述问题
2. 保持章节主题和结构
3. 提升内容质量和深度
4. 使用Markdown格式
5. 不要输出章节标题

直接输出重写后的章节正文内容。`

	return prompt
}
