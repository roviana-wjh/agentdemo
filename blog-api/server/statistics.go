package server

import (
	"awesomeProject1/model"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type blogStatistics struct {
	ArticleCount int64 `json:"article_count"`
	TotalViews   int64 `json:"total_views"`
	CommentCount int64 `json:"comment_count"`
}

type blogStatisticsRepository interface {
	GetPublishedBlogStatistics(ctx context.Context) (blogStatistics, error)
}

type gormBlogStatisticsRepository struct{}

var blogStatisticsRepo blogStatisticsRepository = gormBlogStatisticsRepository{}

func (gormBlogStatisticsRepository) GetPublishedBlogStatistics(ctx context.Context) (blogStatistics, error) {
	var result blogStatistics
	if err := model.DB.WithContext(ctx).
		Model(&model.Article{}).
		Select("COUNT(*) AS article_count, COALESCE(SUM(view_count), 0) AS total_views").
		Where("status = ?", "published").
		Scan(&result).Error; err != nil {
		return blogStatistics{}, err
	}
	if err := model.DB.WithContext(ctx).
		Model(&model.Comment{}).
		Joins("JOIN articles ON articles.id = comments.article_id").
		Where("articles.status = ? AND articles.deleted_at IS NULL", "published").
		Count(&result.CommentCount).Error; err != nil {
		return blogStatistics{}, err
	}
	return result, nil
}

// GetBlogStatisticsTool returns aggregate data for published articles only.
func GetBlogStatisticsTool(c *gin.Context) {
	result, err := blogStatisticsRepo.GetPublishedBlogStatistics(c.Request.Context())
	if err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to get blog statistics")
		return
	}
	toolData(c, result)
}
