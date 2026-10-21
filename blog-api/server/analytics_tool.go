package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetHotArticlesTool ranks published articles by view count.
func GetHotArticlesTool(c *gin.Context) {
	limit, ok := toolLimit(c)
	if !ok {
		return
	}
	articles, err := internalToolsRepository.GetHotPublishedArticles(c.Request.Context(), limit)
	if err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to get hot articles")
		return
	}
	toolData(c, articleResults(articles))
}
