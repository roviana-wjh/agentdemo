package server

import (
	"awesomeProject1/model"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var errDraftNotFound = errors.New("draft not found")

type draftToolRepository interface {
	CreateDraft(ctx context.Context, article *model.Article) error
	UpdateOwnDraft(ctx context.Context, authorID, articleID uint, title, content, image string) (model.Article, error)
}

type gormDraftToolRepository struct{}

var draftToolsRepository draftToolRepository = gormDraftToolRepository{}

func (gormDraftToolRepository) CreateDraft(ctx context.Context, article *model.Article) error {
	return model.DB.WithContext(ctx).Create(article).Error
}

func (gormDraftToolRepository) UpdateOwnDraft(ctx context.Context, authorID, articleID uint, title, content, image string) (model.Article, error) {
	result := model.DB.WithContext(ctx).Model(&model.Article{}).
		Where("id = ? AND author_id = ? AND status = ?", articleID, authorID, "draft").
		Updates(map[string]any{
			"title":         title,
			"content":       content,
			"image":         image,
			"version":       gorm.Expr("version + 1"),
			"content_hash":  hashArticleContent(title, content, image),
			"review_status": "not_reviewed",
			"review_hash":   "",
			"reviewed_at":   nil,
		})
	if result.Error != nil {
		return model.Article{}, result.Error
	}
	if result.RowsAffected != 1 {
		return model.Article{}, errDraftNotFound
	}

	var article model.Article
	if err := model.DB.WithContext(ctx).
		Where("id = ? AND author_id = ? AND status = ?", articleID, authorID, "draft").
		First(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Article{}, errDraftNotFound
		}
		return model.Article{}, err
	}
	return article, nil
}

type createArticleDraftRequest struct {
	AuthorID uint   `json:"author_id" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Image    string `json:"image"`
}

type updateArticleDraftRequest struct {
	AuthorID  uint   `json:"author_id" binding:"required"`
	ArticleID uint   `json:"article_id" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Content   string `json:"content" binding:"required"`
	Image     string `json:"image"`
}

type articleDraftResult struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Version uint   `json:"version"`
	Status  string `json:"status"`
}

func CreateArticleDraftTool(c *gin.Context) {
	var req createArticleDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil || !validDraftFields(req.AuthorID, req.Title, req.Content) {
		toolError(c, http.StatusBadRequest, "invalid_argument", "author_id, title and content are required")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	article := model.Article{
		Title:        req.Title,
		Content:      req.Content,
		Image:        req.Image,
		AuthorID:     req.AuthorID,
		Status:       "draft",
		Version:      1,
		ContentHash:  hashArticleContent(req.Title, req.Content, req.Image),
		IndexStatus:  IndexStatusNotIndexed,
		ReviewStatus: "not_reviewed",
	}
	if err := draftToolsRepository.CreateDraft(c.Request.Context(), &article); err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to create article draft")
		return
	}
	toolData(c, draftResult(article))
}

func UpdateArticleDraftTool(c *gin.Context) {
	var req updateArticleDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ArticleID == 0 || !validDraftFields(req.AuthorID, req.Title, req.Content) {
		toolError(c, http.StatusBadRequest, "invalid_argument", "author_id, article_id, title and content are required")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	article, err := draftToolsRepository.UpdateOwnDraft(c.Request.Context(), req.AuthorID, req.ArticleID, req.Title, req.Content, req.Image)
	if errors.Is(err, errDraftNotFound) {
		toolError(c, http.StatusNotFound, "draft_not_found", "draft not found or not editable by this author")
		return
	}
	if err != nil {
		toolError(c, http.StatusInternalServerError, "internal_error", "failed to update article draft")
		return
	}
	if article.Status != "draft" {
		toolError(c, http.StatusConflict, "not_a_draft", "article is not a draft")
		return
	}
	toolData(c, draftResult(article))
}

func validDraftFields(authorID uint, title, content string) bool {
	return authorID != 0 && strings.TrimSpace(title) != "" && strings.TrimSpace(content) != ""
}

func draftResult(article model.Article) articleDraftResult {
	return articleDraftResult{ID: article.ID, Title: article.Title, Version: article.Version, Status: "draft"}
}
