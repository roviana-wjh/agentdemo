package server

import (
	"awesomeProject1/model"
	"awesomeProject1/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SearchArticle(c *gin.Context) {
	fmt.Println("SearchArticle")
	q := c.Query("q")
	fmt.Println("search q =", q)
	if q == "" {
		c.JSON(200, gin.H{
			"list": []interface{}{},
		})
		return
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  q,
				"fields": []string{"title", "content"},
			},
		},
	}

	body, _ := json.Marshal(query)

	res, err := utils.ES.Search(
		utils.ES.Search.WithIndex("articles"),
		utils.ES.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		c.JSON(500, gin.H{"msg": err.Error()})
		return
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		c.JSON(500, gin.H{"msg": err.Error()})
		return
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println("ES raw result:\n", string(b))

	hitsWrapper := result["hits"].(map[string]interface{})
	hits := hitsWrapper["hits"].([]interface{})

	list := make([]interface{}, 0)
	for _, hit := range hits {
		src := hit.(map[string]interface{})["_source"]
		list = append(list, src)
	}

	c.JSON(200, gin.H{
		"list": list,
	})
}

func PublishArticle(c *gin.Context) {
	userID := c.MustGet("userId").(uint)

	var req struct {
		ID uint `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}

	var article model.Article
	if err := model.DB.Where("id = ? AND author_id = ?", req.ID, userID).First(&article).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "文章不存在或无权发布"})
		return
	}
	if !articleReviewIsCurrent(article) {
		c.JSON(http.StatusConflict, gin.H{
			"code": "review_required",
			"msg":  "文章尚未通过审核，或审核后内容已发生变化",
		})
		return
	}
	wasPublished := article.Status == "published"
	if article.Version == 0 {
		article.Version = 1
	} else if wasPublished {
		article.Version++
	}
	now := time.Now()
	article.Status = "published"
	article.PublishedAt = &now
	article.ContentHash = hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
	article.IndexStatus = IndexStatusPending
	article.IndexError = ""
	if err := model.DB.Model(&article).Updates(map[string]interface{}{
		"status":       article.Status,
		"published_at": article.PublishedAt,
		"version":      article.Version,
		"content_hash": article.ContentHash,
		"index_status": article.IndexStatus,
		"index_error":  "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "发布失败", "error": err.Error()})
		return
	}

	if err := syncArticleIndex(c.Request.Context(), article, defaultAgentClient); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":          "文章已发布，但 AI 索引失败，可稍后重试",
			"id":           article.ID,
			"index_status": IndexStatusFailed,
			"index_error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "发布成功", "id": article.ID, "index_status": IndexStatusSuccess})
}

func ReindexArticle(c *gin.Context) {
	userID := c.MustGet("userId").(uint)
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || articleID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "无效文章ID"})
		return
	}
	var article model.Article
	if err := model.DB.Where("id = ? AND author_id = ? AND status = ?", articleID, userID, "published").First(&article).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "已发布文章不存在或无权操作"})
		return
	}
	if article.Version == 0 {
		article.Version = 1
	} else {
		article.Version++
	}
	article.ContentHash = hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
	if err := model.DB.Model(&article).Updates(map[string]any{
		"index_status": IndexStatusPending,
		"index_error":  "",
		"version":      article.Version,
		"content_hash": article.ContentHash,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "更新索引状态失败"})
		return
	}
	article.IndexStatus = IndexStatusPending
	article.IndexError = ""
	if err := syncArticleIndex(c.Request.Context(), article, defaultAgentClient); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"msg": "重新索引失败", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "重新索引成功", "index_status": IndexStatusSuccess})
}

func SavePost(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(401, gin.H{
			"msg": "未登录",
		})
		return
	}
	var req struct {
		Content string   `json:"content" `
		Image   string   `json:"image" `
		Title   string   `json:"title" `
		ID      uint     `json:"id"`
		Tags    []string `json:"tags"`
	}
	if err := c.ShouldBind(&req); err != nil {
		log.Println("bind error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := userId.(uint)
	tags := normalizeTags(req.Tags)
	contentHash := hashArticleContentWithTags(req.Title, req.Content, req.Image, tags)
	if req.ID == 0 {
		article := model.Article{
			Title:        req.Title,
			Content:      req.Content,
			Image:        req.Image,
			Tags:         tags,
			AuthorID:     userID,
			Status:       "draft",
			Version:      1,
			ContentHash:  contentHash,
			IndexStatus:  IndexStatusNotIndexed,
			ReviewStatus: "not_reviewed",
		}
		if err := model.DB.Create(&article).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "草稿保存失败", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"msg":     "草稿已保存",
			"id":      article.ID,
			"version": article.Version,
		})
		return
	}

	var article model.Article
	if err := model.DB.Where("id = ? AND author_id = ?", req.ID, userID).First(&article).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"msg": "文章不存在或无权编辑"})
		return
	}
	if article.Version == 0 {
		article.Version = 1
	}
	contentChanged := article.ContentHash != contentHash
	if article.Status == "published" && contentChanged {
		c.JSON(http.StatusConflict, gin.H{
			"code": "published_article_requires_draft",
			"msg":  "已发布文章不能直接修改正文，请创建草稿并重新审核后发布",
		})
		return
	}
	if contentChanged {
		article.Version++
	}
	updates := map[string]any{
		"title":        req.Title,
		"content":      req.Content,
		"image":        req.Image,
		"tags":         tags,
		"version":      article.Version,
		"content_hash": contentHash,
	}
	if contentChanged {
		updates["review_status"] = "not_reviewed"
		updates["review_hash"] = ""
		updates["reviewed_at"] = nil
	}
	needsIndex := article.Status == "published" && contentChanged
	if needsIndex {
		updates["index_status"] = IndexStatusPending
		updates["index_error"] = ""
	}
	if err := model.DB.Model(&article).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "文章更新失败", "error": err.Error()})
		return
	}
	article.Title = req.Title
	article.Content = req.Content
	article.Image = req.Image
	article.Tags = tags
	article.ContentHash = contentHash
	if needsIndex {
		article.IndexStatus = IndexStatusPending
		if err := syncArticleIndex(c.Request.Context(), article, defaultAgentClient); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"msg":          "文章已更新，但 AI 索引失败，可稍后重试",
				"id":           article.ID,
				"version":      article.Version,
				"index_status": IndexStatusFailed,
				"index_error":  err.Error(),
			})
			return
		}
		article.IndexStatus = IndexStatusSuccess
	}
	message := "草稿更新成功"
	if article.Status == "published" {
		message = "文章更新成功"
	}
	c.JSON(http.StatusOK, gin.H{
		"msg":          message,
		"id":           article.ID,
		"version":      article.Version,
		"index_status": article.IndexStatus,
	})
}
func DraftPost(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(401, gin.H{
			"msg": "未登录",
		})
	}
	var drafts []model.Article
	model.DB.Where("author_id = ? AND status='draft'", userId).Order("created_at desc").Find(&drafts)
	c.JSON(200, gin.H{
		"list": drafts,
	})
}
func GetPost(c *gin.Context) {
	id := c.Query("id")
	var article model.Article
	if err := model.DB.First(&article, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "文章不存在",
		})
		return
	}
	model.DB.Model(&article).Update("view_count", gorm.Expr("view_count + ?", 1))
	c.JSON(http.StatusOK, gin.H{
		"article": article,
	})
}
