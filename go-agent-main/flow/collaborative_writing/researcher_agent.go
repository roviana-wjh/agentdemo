package collaborative_writing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-agent/tool/blog_tools"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ResearcherAgent 研究员Agent - 负责搜索和总结相关知识
type ResearcherAgent struct {
	model      model.BaseChatModel
	searchTool tool.InvokableTool
}

// NewResearcherAgent 创建研究员Agent
func NewResearcherAgent(chatModel model.BaseChatModel) (*ResearcherAgent, error) {
	// 获取博客搜索工具
	tools, err := blog_tools.GetTools()
	if err != nil {
		return nil, fmt.Errorf("get blog tools: %w", err)
	}

	var searchTool tool.InvokableTool
	for _, t := range tools {
		info, _ := t.Info(context.Background())
		if info.Name == "search_article" {
			searchTool = t.(tool.InvokableTool)
			break
		}
	}

	if searchTool == nil {
		return nil, fmt.Errorf("search_article tool not found")
	}

	return &ResearcherAgent{
		model:      chatModel,
		searchTool: searchTool,
	}, nil
}

// Research 执行研究任务
func (a *ResearcherAgent) Research(ctx context.Context, topic string, requirements string) (ResearchOutput, error) {
	log.Printf("[ResearcherAgent] 开始研究主题: %s", topic)

	// Step 1: 提取关键词
	keywords, err := a.extractKeywords(ctx, topic, requirements)
	if err != nil {
		return ResearchOutput{}, fmt.Errorf("extract keywords: %w", err)
	}
	log.Printf("[ResearcherAgent] 提取关键词: %v", keywords)

	// Step 2: 搜索相关文章
	articles, err := a.searchArticles(ctx, keywords)
	if err != nil {
		return ResearchOutput{}, fmt.Errorf("search articles: %w", err)
	}
	log.Printf("[ResearcherAgent] 找到 %d 篇相关文章", len(articles))

	// Step 3: 生成研究摘要
	summary, keyFindings, err := a.summarizeResearch(ctx, topic, articles)
	if err != nil {
		return ResearchOutput{}, fmt.Errorf("summarize research: %w", err)
	}

	totalWords := 0
	for _, article := range articles {
		totalWords += len(article.Content)
	}

	log.Printf("[ResearcherAgent] 研究完成，生成摘要 %d 字", len(summary))

	return ResearchOutput{
		Articles:    articles,
		Summary:     summary,
		KeyFindings: keyFindings,
		TotalWords:  totalWords,
		Completed:   true,
	}, nil
}

// extractKeywords 从主题中提取搜索关键词
func (a *ResearcherAgent) extractKeywords(ctx context.Context, topic string, requirements string) ([]string, error) {
	prompt := fmt.Sprintf(`你是技术文章研究专家。分析以下主题，提取3-5个最关键的搜索词。

主题：%s
要求：%s

输出JSON格式：{"keywords": ["关键词1", "关键词2", ...]}

只输出JSON，不要其他内容。`, topic, requirements)

	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是关键词提取专家，只输出JSON格式"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var result struct {
		Keywords []string `json:"keywords"`
	}
	content := extractJSONFromResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 如果解析失败，使用主题本身作为关键词
		log.Printf("[ResearcherAgent] JSON解析失败，使用主题作为关键词: %v", err)
		return []string{topic}, nil
	}

	if len(result.Keywords) == 0 {
		return []string{topic}, nil
	}

	return result.Keywords, nil
}

// searchArticles 搜索相关文章
func (a *ResearcherAgent) searchArticles(ctx context.Context, keywords []string) ([]ArticleReference, error) {
	var allArticles []ArticleReference

	// 对每个关键词搜索
	for _, keyword := range keywords {
		// 构建搜索参数
		searchInput := blog_tools.SearchArticleInput{
			Keyword: keyword,
			Limit:   5, // 每个关键词最多5篇
		}
		inputJSON, _ := json.Marshal(searchInput)

		// 调用搜索工具
		resultStr, err := a.searchTool.InvokableRun(ctx, string(inputJSON))
		if err != nil {
			log.Printf("[ResearcherAgent] 搜索关键词 '%s' 失败: %v", keyword, err)
			continue
		}

		// 解析搜索结果
		var searchResult blog_tools.SearchArticleResult
		if err := json.Unmarshal([]byte(resultStr), &searchResult); err != nil {
			log.Printf("[ResearcherAgent] 解析搜索结果失败: %v", err)
			continue
		}

		// 转换为ArticleReference
		for _, article := range searchResult.Articles {
			allArticles = append(allArticles, ArticleReference{
				ID:      article.ID,
				Title:   article.Title,
				Content: article.Content, // 使用内容
				Tags:    []string{},
				URL:     fmt.Sprintf("/articles/%d", article.ID),
			})
		}
	}

	// 去重（根据ID）
	seen := make(map[uint]bool)
	uniqueArticles := []ArticleReference{}
	for _, article := range allArticles {
		if !seen[article.ID] {
			seen[article.ID] = true
			uniqueArticles = append(uniqueArticles, article)
		}
	}

	// 最多保留10篇
	if len(uniqueArticles) > 10 {
		uniqueArticles = uniqueArticles[:10]
	}

	return uniqueArticles, nil
}

// summarizeResearch 总结研究结果
func (a *ResearcherAgent) summarizeResearch(ctx context.Context, topic string, articles []ArticleReference) (string, []string, error) {
	if len(articles) == 0 {
		return "未找到相关文章，将基于主题进行创作。", []string{}, nil
	}

	// 构建文章列表
	articlesText := ""
	for i, article := range articles {
		articlesText += fmt.Sprintf("\n[%d] 标题：%s\n摘要：%s\n", i+1, article.Title, article.Content)
	}

	prompt := fmt.Sprintf(`你是技术文章研究专家。基于以下搜索到的博客文章，生成一份研究摘要。

主题：%s

相关文章：
%s

请生成：
1. 一段500字以内的研究摘要，概括现有文章的核心观点
2. 3-5个关键发现点

输出JSON格式：
{
  "summary": "研究摘要文本...",
  "key_findings": ["发现1", "发现2", ...]
}

只输出JSON，不要其他内容。`, topic, articlesText)

	response, err := a.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是研究总结专家，只输出JSON格式"),
		schema.UserMessage(prompt),
	})
	if err != nil {
		return "", nil, err
	}

	// 解析JSON
	var result struct {
		Summary     string   `json:"summary"`
		KeyFindings []string `json:"key_findings"`
	}
	content := extractJSONFromResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// 解析失败，返回简单摘要
		log.Printf("[ResearcherAgent] JSON解析失败: %v", err)
		simpleSummary := fmt.Sprintf("找到%d篇相关文章，涵盖了%s相关的内容。", len(articles), topic)
		return simpleSummary, []string{}, nil
	}

	return result.Summary, result.KeyFindings, nil
}

// extractJSONFromResponse 从响应中提取JSON
func extractJSONFromResponse(content string) string {
	content = trimMarkdownCodeBlock(content)
	return content
}

// trimMarkdownCodeBlock 移除markdown代码块标记
func trimMarkdownCodeBlock(s string) string {
	s = trimPrefix(s, "```json")
	s = trimPrefix(s, "```")
	s = trimSuffix(s, "```")
	return trimSpace(s)
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
