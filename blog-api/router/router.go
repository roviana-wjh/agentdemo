package router

import (
	"awesomeProject1/middleware"
	comment "awesomeProject1/server"
	post "awesomeProject1/server"
	upload "awesomeProject1/server"
	visitor "awesomeProject1/server"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.StaticFS("/static", http.Dir("static"))
	registerBlogWeb(router)
	api := router.Group("/api")
	{
		api.POST("/login", visitor.Login)
		api.GET("/article/list", visitor.Home)
		api.GET("/article/:id", post.GetPost)
		api.GET("/comment/list", post.GetComment)
		api.GET("/article/search", post.SearchArticle)
	}
	api2 := router.Group("/api", middleware.JwtAuthMiddleware)
	{
		api2.POST("/logout", visitor.Logout)
		api2.POST("/article/publish", post.PublishArticle)
		api2.POST("/article/:id/review", post.ReviewArticle)
		api2.GET("/article/:id/review", post.GetArticleReview)
		api2.POST("/article/:id/reindex", post.ReindexArticle)
		api2.POST("/article/save", post.SavePost)
		api2.POST("/upload", upload.UploadImage)
		api2.POST("comment/create", comment.CreateComment)
		api2.GET("article/drafts", post.DraftPost)
		api2.POST("/ai/chat/stream", post.AIChatStream)
		api2.POST("/ai/assistant/stream", post.UnifiedAIChatStream)
		api2.POST("/ai/writing/stream", post.AIWritingStream)
		api2.POST("/ai/drafts/save", post.SaveAIWritingDraft)
		api2.POST("/ai/articles/summary/stream", post.AIArticleSummary)
		api2.POST("/ai/comments/analyze/stream", post.AICommentAnalysis)
		api2.POST("/ai/analytics/stream", post.AISimpleAnalytics)
	}
	internalTools := router.Group("/internal/v1/tools", middleware.InternalAuthMiddleware)
	{
		internalTools.GET("/search_article", post.SearchArticleTool)
		internalTools.GET("/get_article_detail", post.GetArticleDetailTool)
		internalTools.GET("/get_hot_articles", post.GetHotArticlesTool)
		internalTools.GET("/get_recent_comments", post.GetRecentCommentsTool)
		internalTools.GET("/get_hot_tags", post.GetHotTagsTool)
		internalTools.GET("/get_blog_statistics", post.GetBlogStatisticsTool)
		internalTools.POST("/create_article_draft", post.CreateArticleDraftTool)
		internalTools.POST("/update_article_draft", post.UpdateArticleDraftTool)
	}
	return router
}

func registerBlogWeb(router *gin.Engine) {
	dir := strings.TrimSpace(os.Getenv("BLOG_WEB_DIR"))
	if dir == "" {
		dir = filepath.Join("..", "blog-web")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	if info, err := os.Stat(filepath.Join(abs, "index.html")); err != nil || info.IsDir() {
		return
	}
	router.StaticFS("/web", http.Dir(abs))
	router.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/web/") })
}
