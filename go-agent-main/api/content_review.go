package api

import (
	"go-agent/flow"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func ReviewArticle(c *gin.Context) {
	if !validBlogInternalToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req flow.ContentReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ArticleID == 0 ||
		strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" ||
		strings.TrimSpace(req.ContentHash) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "article_id, title, content and content_hash are required"})
		return
	}
	agent, err := flow.GetContentReviewAgent()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	result := agent.Review(c.Request.Context(), req)
	c.JSON(http.StatusOK, gin.H{"data": result})
}
