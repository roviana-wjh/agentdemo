package server

import (
	agentclient "awesomeProject1/client"
	"awesomeProject1/model"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	reviewStatusNotReviewed = "not_reviewed"
	reviewStatusApproved    = "approved"
)

type articleReviewRepository interface {
	FindOwnedArticle(ctx context.Context, articleID, authorID uint) (model.Article, error)
	SaveReview(ctx context.Context, article model.Article, result agentclient.ArticleReviewResult, reviewedAt time.Time) error
	LatestReview(ctx context.Context, articleID, authorID uint) (model.Article, model.ArticleReview, error)
}

type gormArticleReviewRepository struct{}

var reviewsRepository articleReviewRepository = gormArticleReviewRepository{}

func (gormArticleReviewRepository) FindOwnedArticle(ctx context.Context, articleID, authorID uint) (model.Article, error) {
	var article model.Article
	err := model.DB.WithContext(ctx).Where("id = ? AND author_id = ?", articleID, authorID).First(&article).Error
	return article, err
}

func (gormArticleReviewRepository) SaveReview(ctx context.Context, article model.Article, result agentclient.ArticleReviewResult, reviewedAt time.Time) error {
	categories, err := json.Marshal(result.Categories)
	if err != nil {
		return err
	}
	reasons, err := json.Marshal(result.Reasons)
	if err != nil {
		return err
	}
	suggestions, err := json.Marshal(result.Suggestions)
	if err != nil {
		return err
	}
	ruleHits, err := json.Marshal(result.RuleHits)
	if err != nil {
		return err
	}

	return model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		review := model.ArticleReview{
			ArticleID: article.ID, ArticleVersion: article.Version, ContentHash: result.ContentHash,
			Status: result.Status, RiskLevel: result.RiskLevel, Score: result.Score,
			CategoriesJSON: string(categories), ReasonsJSON: string(reasons),
			SuggestionsJSON: string(suggestions), RuleHitsJSON: string(ruleHits),
		}
		if err := tx.Create(&review).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"review_status": result.Status,
			"review_hash":   result.ContentHash,
			"reviewed_at":   reviewedAt,
		}
		update := tx.Model(&model.Article{}).
			Where("id = ? AND author_id = ? AND version = ? AND content_hash = ?", article.ID, article.AuthorID, article.Version, result.ContentHash).
			Updates(updates)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return errors.New("article changed while review was running")
		}
		return nil
	})
}

func (gormArticleReviewRepository) LatestReview(ctx context.Context, articleID, authorID uint) (model.Article, model.ArticleReview, error) {
	article, err := (gormArticleReviewRepository{}).FindOwnedArticle(ctx, articleID, authorID)
	if err != nil {
		return model.Article{}, model.ArticleReview{}, err
	}
	var review model.ArticleReview
	err = model.DB.WithContext(ctx).Where("article_id = ?", articleID).Order("id DESC").First(&review).Error
	return article, review, err
}

func ReviewArticle(c *gin.Context) {
	userID := c.MustGet("userId").(uint)
	articleID, err := parseArticleID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "无效的文章 ID"})
		return
	}
	article, err := reviewsRepository.FindOwnedArticle(c.Request.Context(), articleID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"msg": "文章不存在或无权审核"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "读取文章失败"})
		return
	}
	if article.Status != "draft" {
		c.JSON(http.StatusConflict, gin.H{"msg": "仅草稿可以发起审核"})
		return
	}
	contentHash := hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
	result, err := defaultAgentClient.ReviewArticle(c.Request.Context(), agentclient.ArticleReviewRequest{
		ArticleID: article.ID, Version: article.Version, Source: "draft",
		Title: article.Title, Content: article.Content, Tags: []string(article.Tags), ContentHash: contentHash,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"msg": "AI 审核服务调用失败", "error": err.Error()})
		return
	}
	now := time.Now()
	if err := reviewsRepository.SaveReview(c.Request.Context(), article, result, now); err != nil {
		c.JSON(http.StatusConflict, gin.H{"msg": "文章在审核期间发生变化，请重新审核", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "reviewed_at": now})
}

func GetArticleReview(c *gin.Context) {
	userID := c.MustGet("userId").(uint)
	articleID, err := parseArticleID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "无效的文章 ID"})
		return
	}
	article, review, err := reviewsRepository.LatestReview(c.Request.Context(), articleID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if article.ID == 0 {
			c.JSON(http.StatusNotFound, gin.H{"msg": "文章不存在或无权查看"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": reviewStatusNotReviewed}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "读取审核结果失败"})
		return
	}
	currentHash := hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
	if article.ReviewStatus == reviewStatusNotReviewed || review.ContentHash != currentHash {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": reviewStatusNotReviewed}})
		return
	}
	result, err := storedReviewResult(article, review)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "解析审核结果失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "reviewed_at": review.CreatedAt})
}

func articleReviewIsCurrent(article model.Article) bool {
	if article.ReviewStatus != reviewStatusApproved || article.ReviewHash == "" {
		return false
	}
	return article.ReviewHash == hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
}

func parseArticleID(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid article id")
	}
	return uint(id), nil
}

func storedReviewResult(article model.Article, review model.ArticleReview) (agentclient.ArticleReviewResult, error) {
	result := agentclient.ArticleReviewResult{
		ArticleID: article.ID, Version: review.ArticleVersion, ContentHash: review.ContentHash,
		Status: review.Status, RiskLevel: review.RiskLevel, Score: review.Score,
	}
	if err := json.Unmarshal([]byte(review.CategoriesJSON), &result.Categories); err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(review.ReasonsJSON), &result.Reasons); err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(review.SuggestionsJSON), &result.Suggestions); err != nil {
		return result, err
	}
	if err := json.Unmarshal([]byte(review.RuleHitsJSON), &result.RuleHits); err != nil {
		return result, err
	}
	return result, nil
}
