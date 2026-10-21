package server

import (
	"awesomeProject1/model"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultToolLimit = 10
	maxToolLimit     = 50
)

type internalToolRepository interface {
	SearchPublishedArticles(ctx context.Context, query string, limit int) ([]internalArticleRecord, error)
	GetPublishedArticle(ctx context.Context, id uint) (internalArticleRecord, error)
	GetHotPublishedArticles(ctx context.Context, limit int) ([]internalArticleRecord, error)
	GetRecentPublishedComments(ctx context.Context, articleID uint, limit int) ([]internalCommentRecord, error)
}

type gormInternalToolRepository struct{}

var internalToolsRepository internalToolRepository = gormInternalToolRepository{}

type internalArticleRecord struct {
	Article      model.Article
	CommentCount int64
}

type internalCommentRecord struct {
	Comment      model.Comment
	ArticleTitle string
}

func (gormInternalToolRepository) SearchPublishedArticles(ctx context.Context, query string, limit int) ([]internalArticleRecord, error) {
	var articles []model.Article
	pattern := "%" + query + "%"
	err := model.DB.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Where("status = ? AND (title LIKE ? OR content LIKE ?)", "published", pattern, pattern).
		Order("published_at DESC").
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	if err != nil {
		return nil, err
	}
	return addArticleCommentCounts(ctx, articles)
}

func (gormInternalToolRepository) GetPublishedArticle(ctx context.Context, id uint) (internalArticleRecord, error) {
	var article model.Article
	err := model.DB.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Where("id = ? AND status = ?", id, "published").
		First(&article).Error
	if err != nil {
		return internalArticleRecord{}, err
	}
	articles, err := addArticleCommentCounts(ctx, []model.Article{article})
	if err != nil {
		return internalArticleRecord{}, err
	}
	return articles[0], nil
}

func (gormInternalToolRepository) GetHotPublishedArticles(ctx context.Context, limit int) ([]internalArticleRecord, error) {
	var articles []model.Article
	err := model.DB.WithContext(ctx).
		Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Where("status = ?", "published").
		Order("view_count DESC").
		Order("published_at DESC").
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	if err != nil {
		return nil, err
	}
	return addArticleCommentCounts(ctx, articles)
}

func (gormInternalToolRepository) GetRecentPublishedComments(ctx context.Context, articleID uint, limit int) ([]internalCommentRecord, error) {
	var comments []model.Comment
	query := model.DB.WithContext(ctx).
		Model(&model.Comment{}).
		Joins("JOIN articles ON articles.id = comments.article_id AND articles.deleted_at IS NULL").
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Where("articles.status = ?", "published")
	if articleID != 0 {
		query = query.Where("comments.article_id = ?", articleID)
	}
	err := query.Order("comments.created_at DESC").
		Order("comments.id DESC").
		Limit(limit).
		Find(&comments).Error
	if err != nil || len(comments) == 0 {
		return nil, err
	}
	articleIDs := make([]uint, 0, len(comments))
	seen := make(map[uint]struct{}, len(comments))
	for _, comment := range comments {
		if _, exists := seen[comment.ArticleID]; !exists {
			seen[comment.ArticleID] = struct{}{}
			articleIDs = append(articleIDs, comment.ArticleID)
		}
	}
	var titles []struct {
		ID    uint
		Title string
	}
	if err := model.DB.WithContext(ctx).Model(&model.Article{}).
		Select("id", "title").
		Where("id IN ? AND status = ?", articleIDs, "published").
		Find(&titles).Error; err != nil {
		return nil, err
	}
	titleByArticleID := make(map[uint]string, len(titles))
	for _, article := range titles {
		titleByArticleID[article.ID] = article.Title
	}
	result := make([]internalCommentRecord, 0, len(comments))
	for _, comment := range comments {
		result = append(result, internalCommentRecord{Comment: comment, ArticleTitle: titleByArticleID[comment.ArticleID]})
	}
	return result, nil
}

func addArticleCommentCounts(ctx context.Context, articles []model.Article) ([]internalArticleRecord, error) {
	if len(articles) == 0 {
		return []internalArticleRecord{}, nil
	}
	articleIDs := make([]uint, 0, len(articles))
	for _, article := range articles {
		articleIDs = append(articleIDs, article.ID)
	}
	var counts []struct {
		ArticleID uint  `gorm:"column:article_id"`
		Count     int64 `gorm:"column:count"`
	}
	if err := model.DB.WithContext(ctx).Model(&model.Comment{}).
		Select("article_id, COUNT(*) AS count").
		Where("article_id IN ?", articleIDs).
		Group("article_id").
		Find(&counts).Error; err != nil {
		return nil, err
	}
	countByArticleID := make(map[uint]int64, len(counts))
	for _, count := range counts {
		countByArticleID[count.ArticleID] = count.Count
	}
	result := make([]internalArticleRecord, 0, len(articles))
	for _, article := range articles {
		result = append(result, internalArticleRecord{Article: article, CommentCount: countByArticleID[article.ID]})
	}
	return result, nil
}

type internalArticle struct {
	ID           uint       `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Image        string     `json:"image,omitempty"`
	Tags         []string   `json:"tags"`
	AuthorID     uint       `json:"author_id"`
	AuthorName   string     `json:"author_name"`
	ViewCount    int        `json:"view_count"`
	CommentCount int64      `json:"comment_count"`
	Version      uint       `json:"version"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

type internalComment struct {
	ID           uint      `json:"id"`
	Content      string    `json:"content"`
	UserID       uint      `json:"user_id"`
	UserName     string    `json:"user_name"`
	ArticleID    uint      `json:"article_id"`
	ArticleTitle string    `json:"article_title"`
	CreatedAt    time.Time `json:"created_at"`
}

func articleResult(record internalArticleRecord) internalArticle {
	article := record.Article
	return internalArticle{
		ID:           article.ID,
		Title:        article.Title,
		Content:      article.Content,
		Image:        article.Image,
		Tags:         append([]string{}, article.Tags...),
		AuthorID:     article.AuthorID,
		AuthorName:   article.Author.Name,
		ViewCount:    article.ViewCount,
		CommentCount: record.CommentCount,
		Version:      article.Version,
		PublishedAt:  article.PublishedAt,
	}
}

func articleResults(articles []internalArticleRecord) []internalArticle {
	result := make([]internalArticle, 0, len(articles))
	for _, article := range articles {
		// Keep a final response-layer guard in addition to the repository
		// predicate so drafts can never be exposed by these read-only tools.
		if article.Article.Status != "published" {
			continue
		}
		result = append(result, articleResult(article))
	}
	return result
}

func commentResults(comments []internalCommentRecord) []internalComment {
	result := make([]internalComment, 0, len(comments))
	for _, record := range comments {
		comment := record.Comment
		result = append(result, internalComment{
			ID:           comment.ID,
			Content:      comment.Content,
			UserID:       comment.UserID,
			UserName:     comment.User.Name,
			ArticleID:    comment.ArticleID,
			ArticleTitle: record.ArticleTitle,
			CreatedAt:    comment.CreatedAt,
		})
	}
	return result
}

func toolLimit(c *gin.Context) (int, bool) {
	raw := strings.TrimSpace(c.Query("limit"))
	if raw == "" {
		return defaultToolLimit, true
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		toolError(c, http.StatusBadRequest, "invalid_argument", "limit must be a positive integer")
		return 0, false
	}
	if limit > maxToolLimit {
		limit = maxToolLimit
	}
	return limit, true
}

func toolUintQuery(c *gin.Context, name string, required bool) (uint, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" && !required {
		return 0, true
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 || uint64(uint(value)) != value {
		toolError(c, http.StatusBadRequest, "invalid_argument", name+" must be a positive integer")
		return 0, false
	}
	return uint(value), true
}

func toolData(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func toolError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
