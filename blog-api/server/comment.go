package server

import (
	"awesomeProject1/model"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func CreateComment(c *gin.Context) {
	log.Println(">>> CreateComment called")
	userID, ok := c.MustGet("userId").(uint)
	if !ok {
		c.JSON(401, gin.H{"msg": "未登录"})
		return
	}
	var req struct {
		ArticleID uint   `json:"article_id" binding:"required"`
		Content   string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("bind error:", err)
		body, _ := c.GetRawData()
		log.Println("raw body:", string(body))
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}

	comment := model.Comment{
		Content:   req.Content,
		UserID:    userID,
		ArticleID: req.ArticleID,
	}
	log.Println("before create:", comment)
	if err := model.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		log.Println("after create err:", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": "评论成功",
	})
}

func GetComment(c *gin.Context) {
	articleID := c.Query("article_id")
	var comments []model.Comment
	model.DB.Preload("User").Model(&model.Comment{}).Where("article_id = ?", articleID).Order("created_at asc").Find(&comments)
	for _, c := range comments {
		log.Println("comment:", c.ID, "user:", c.User.ID, c.User.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
	})
}
