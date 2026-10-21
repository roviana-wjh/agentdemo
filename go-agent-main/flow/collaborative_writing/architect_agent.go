package collaborative_writing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ArchitectAgent 架构师Agent - 负责设计文章结构和大纲
type ArchitectAgent struct {
	model model.BaseChatModel
}

// NewArchitectAgent 创建架构师Agent
func NewArchitectAgent(chatModel model.BaseChatModel) *ArchitectAgent {
	return &ArchitectAgent{
		model: chatModel,
	}
}

// DesignOutline 设计文章大纲
func (a *ArchitectAgent) DesignOutline(
	ctx context.Context,
	topic string,
	requirements string,
	research ResearchOutput,
	targetWords int,
	difficulty string,
	tone string,
) (ArchitectureOutput, error) {
	log.Printf("[ArchitectAgent] 开始设计文章大纲: %s", topic)

	// 默认参数
	if targetWords == 0 {
		targetWords = 2000
	}
	if difficulty == "" {
		difficulty = "intermediate"
	}
	if tone == "" {
		tone = "technical"
	}

	// 构建Prompt
	prompt := a.buildPrompt(topic, requirements, research, targetWords, difficulty, tone)

	// 调用LLM生成大纲
	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是资深技术文章架构师，擅长设计清晰、有逻辑的文章结构。只输出JSON格式。"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return ArchitectureOutput{}, fmt.Errorf("generate outline: %w", err)
	}

	// 解析JSON
	var outline ArticleOutline
	content := extractJSONFromResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return ArchitectureOutput{}, fmt.Errorf("parse outline JSON: %w (content: %s)", err, content)
	}

	// 补充字段
	outline.Difficulty = difficulty
	outline.Tone = tone
	if outline.TotalWords == 0 {
		outline.TotalWords = targetWords
	}

	// 生成设计理由
	rationale := a.generateRationale(ctx, topic, outline)

	log.Printf("[ArchitectAgent] 大纲设计完成: %d个章节，目标%d字", len(outline.Sections), outline.TotalWords)

	return ArchitectureOutput{
		Outline:     outline,
		Rationale:   rationale,
		TargetWords: targetWords,
		Completed:   true,
	}, nil
}

// buildPrompt 构建Prompt
func (a *ArchitectAgent) buildPrompt(
	topic string,
	requirements string,
	research ResearchOutput,
	targetWords int,
	difficulty string,
	tone string,
) string {
	researchContext := ""
	if research.Completed && len(research.KeyFindings) > 0 {
		researchContext = fmt.Sprintf(`

研究摘要：
%s

关键发现：
`, research.Summary)
		for i, finding := range research.KeyFindings {
			researchContext += fmt.Sprintf("%d. %s\n", i+1, finding)
		}
	}

	difficultyDesc := map[string]string{
		"beginner":     "初学者（基础概念，通俗易懂）",
		"intermediate": "中级开发者（有一定基础，深入实践）",
		"advanced":     "高级开发者（深度原理，最佳实践）",
	}[difficulty]

	toneDesc := map[string]string{
		"formal":    "正式、学术性",
		"casual":    "轻松、口语化",
		"technical": "技术性、专业",
	}[tone]

	return fmt.Sprintf(`设计一篇技术文章的详细大纲。

主题：%s
要求：%s
目标字数：%d字
难度级别：%s
语气风格：%s
%s

请设计包含5-7个章节的文章大纲。每个章节需要：
- 清晰的章节标题（简洁有力）
- 章节描述（100字内，说明本章节核心内容）
- 3-5个关键要点（具体、可操作）
- 预计字数（合理分配，总和=%d字）

设计原则：
1. 结构清晰：从基础→深入→实践，符合认知规律
2. 逻辑递进：章节间有明确的承接关系
3. 理论+实践：既有原理讲解，又有代码示例
4. 重点突出：核心章节字数可以多一些
5. 完整闭环：有引言（10%%）、正文（80%%）、总结（10%%）

输出JSON格式：
{
  "title": "文章标题（吸引人，准确概括主题）",
  "subtitle": "副标题（可选，补充说明）",
  "sections": [
    {
      "order": 1,
      "title": "章节标题",
      "description": "章节描述（说明本章要讲什么，为什么重要）",
      "key_points": [
        "要点1（具体明确，不要太宽泛）",
        "要点2",
        "要点3"
      ],
      "target_words": 300
    }
  ],
  "total_words": %d
}

注意：
- title要有吸引力，避免平淡无奇
- 第一章应该是引言/背景，最后一章应该是总结/展望
- key_points要具体，不要"介绍XX"这种模糊表述
- target_words要合理分配，核心章节可以400-500字

只输出JSON，不要其他内容。`, topic, requirements, targetWords, difficultyDesc, toneDesc, researchContext, targetWords, targetWords)
}

// generateRationale 生成设计理由
func (a *ArchitectAgent) generateRationale(ctx context.Context, topic string, outline ArticleOutline) string {
	prompt := fmt.Sprintf(`解释以下文章大纲的设计思路（100字以内）。

主题：%s
大纲标题：%s
章节数：%d

设计思路：`, topic, outline.Title, len(outline.Sections))

	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是技术写作专家"),
		schema.UserMessage(prompt),
	})

	if err != nil {
		return "基于主题和研究结果设计的结构化大纲"
	}

	return response.Content
}
