package api

import (
	"context"
	"errors"
	"go-agent/rag/article"
	"net/http"

	"github.com/gin-gonic/gin"
)

type articleIndexer interface {
	Index(ctx context.Context, snapshot article.Snapshot) (article.IndexResult, error)
}

func IndexArticle(c *gin.Context) {
	service := article.GetDefaultIndexService()
	if service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "article index service is not initialized"})
		return
	}
	indexArticleHandler(service)(c)
}

func indexArticleHandler(service articleIndexer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var snapshot article.Snapshot
		if err := c.ShouldBindJSON(&snapshot); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid article snapshot: " + err.Error()})
			return
		}
		if err := snapshot.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := service.Index(c.Request.Context(), snapshot)
		if err != nil {
			var indexErr *article.IndexError
			if errors.As(err, &indexErr) {
				c.JSON(http.StatusBadGateway, gin.H{
					"success": false,
					"error":   err.Error(),
					"result":  indexErr.Result,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"chunk_count": len(result.ChunkIDs),
			"result":      result,
		})
	}
}
