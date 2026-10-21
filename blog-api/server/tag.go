package server

import (
	"awesomeProject1/model"
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type hotTagsRepository interface {
	ListPublishedArticleTags(ctx context.Context) ([]model.Tags, error)
}

type gormHotTagsRepository struct{}

var hotTagsRepo hotTagsRepository = gormHotTagsRepository{}

func (gormHotTagsRepository) ListPublishedArticleTags(ctx context.Context) ([]model.Tags, error) {
	var articles []struct {
		Tags model.Tags `gorm:"column:tags"`
	}
	err := model.DB.WithContext(ctx).
		Model(&model.Article{}).
		Select("tags").
		Where("status = ?", "published").
		Find(&articles).Error
	if err != nil {
		return nil, err
	}
	result := make([]model.Tags, 0, len(articles))
	for _, article := range articles {
		result = append(result, article.Tags)
	}
	return result, nil
}

type hotTag struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// GetHotTagsTool returns tags used by published articles, ranked by article
// count. A repeated tag in one article is counted once.
func GetHotTagsTool(c *gin.Context) {
	limit, ok := toolLimit(c)
	if !ok {
		return
	}
	articleTags, err := hotTagsRepo.ListPublishedArticleTags(c.Request.Context())
	if err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to get hot tags")
		return
	}
	counts := make(map[string]int64)
	labels := make(map[string]string)
	for _, tags := range articleTags {
		seen := make(map[string]struct{}, len(tags))
		for _, raw := range tags {
			tag := strings.TrimSpace(raw)
			if tag == "" {
				continue
			}
			key := strings.ToLower(tag)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			if _, exists := labels[key]; !exists {
				labels[key] = tag
			}
			counts[key]++
		}
	}
	result := make([]hotTag, 0, len(counts))
	for key, count := range counts {
		result = append(result, hotTag{Tag: labels[key], Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Tag < result[j].Tag
		}
		return result[i].Count > result[j].Count
	})
	if len(result) > limit {
		result = result[:limit]
	}
	toolData(c, result)
}

func normalizeTags(values []string) model.Tags {
	result := make(model.Tags, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		tag := strings.TrimSpace(value)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}
