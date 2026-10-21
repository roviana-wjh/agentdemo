package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchArticleTool searches published articles only.
func SearchArticleTool(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		toolError(c, http.StatusBadRequest, "invalid_argument", "q must not be empty")
		return
	}
	limit, ok := toolLimit(c)
	if !ok {
		return
	}
	articles, err := internalToolsRepository.SearchPublishedArticles(c.Request.Context(), query, limit)
	if err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to search articles")
		return
	}
	toolData(c, articleResults(articles))
}

// GetArticleDetailTool returns one published article with its full content.
func GetArticleDetailTool(c *gin.Context) {
	id, ok := toolUintQuery(c, "id", true)
	if !ok {
		return
	}
	article, err := internalToolsRepository.GetPublishedArticle(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			toolError(c, http.StatusNotFound, "not_found", "published article not found")
			return
		}
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to get article")
		return
	}
	if article.Article.Status != "published" {
		toolError(c, http.StatusNotFound, "not_found", "published article not found")
		return
	}
	toolData(c, articleResult(article))
}
