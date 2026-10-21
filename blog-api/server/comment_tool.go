package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRecentCommentsTool returns comments belonging to published articles. An
// optional article_id narrows the result to one published article.
func GetRecentCommentsTool(c *gin.Context) {
	articleID, ok := toolUintQuery(c, "article_id", false)
	if !ok {
		return
	}
	limit, ok := toolLimit(c)
	if !ok {
		return
	}
	comments, err := internalToolsRepository.GetRecentPublishedComments(c.Request.Context(), articleID, limit)
	if err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to get recent comments")
		return
	}
	toolData(c, commentResults(comments))
}
