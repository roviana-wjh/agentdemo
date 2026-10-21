package api

import (
	"go-agent/config"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func Run() {
	r := gin.Default()
	r.MaxMultipartMemory = 50 << 20

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("failed to get working directory: %v", err)
		workDir = "."
	}

	htmlPath := absOrFallback(filepath.Join(workDir, "chat_test.html"))
	ragIndexPath := absOrFallback(filepath.Join(workDir, "rag_index.html"))
	ragAskPath := absOrFallback(filepath.Join(workDir, "rag_ask.html"))

	hasChatTest := registerStaticPage(r, "/chat_test.html", htmlPath, "chat test page")
	registerStaticPage(r, "/rag_index.html", ragIndexPath, "RAG index page")
	hasRagAsk := registerStaticPage(r, "/rag_ask.html", ragAskPath, "RAG ask page")

	if hasRagAsk {
		r.GET("/", func(c *gin.Context) { c.Redirect(302, "/rag_ask.html") })
	} else if hasChatTest {
		r.GET("/", func(c *gin.Context) { c.Redirect(302, "/chat_test.html") })
	} else {
		r.GET("/", func(c *gin.Context) {
			c.String(200, "No page files found in %s", workDir)
		})
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "server is running",
		})
	})

	r.POST("/api/document/insert", InsertDocument)
	r.POST("/api/rag/insert", InsertDocument)
	r.POST("/internal/v1/index/articles", IndexArticle)

	r.POST("/api/chat/test", ChatGenerate)
	r.POST("/api/chat/test/stream", ChatStream)
	r.POST("/v1/chat/stream", ChatStreamV1)

	r.POST("/api/rag/ask", RAGAsk)
	r.POST("/api/rag/chat/stream", RAGChatStream)
	r.POST("/v1/rag/chat/stream", RAGChatStream)
	r.POST("/v1/blog/chat/stream", BlogChatStream)
	r.POST("/v1/blog/assistant/stream", UnifiedAssistantStream)
	r.POST("/v1/unified/chat/stream", UnifiedChatStream) // 新的统一端点
	r.POST("/v1/blog/writing/stream", WritingStream)
	r.POST("/v1/blog/drafts/save", SaveWritingDraft)
	r.POST("/v1/blog/articles/review", ReviewArticle)
	r.POST("/v1/blog/articles/summary/stream", ArticleSummaryStream)
	r.POST("/v1/blog/comments/analyze/stream", CommentAnalysisStream)
	r.POST("/v1/blog/analytics/stream", SimpleAnalyticsStream)

	// 多Agent协作写作
	r.POST("/v1/collaborative/writing", CollaborativeWriting)
	r.POST("/v1/collaborative/writing/stream", CollaborativeWritingStream)

	r.GET("/api/milvus/collections", ListMilvusCollections)
	r.DELETE("/api/milvus/collections/:name", DeleteMilvusCollection)

	if err := r.Run(serverAddress()); err != nil {
		log.Fatalf("run fail: %v", err)
	}
}

func absOrFallback(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func registerStaticPage(r *gin.Engine, route string, path string, label string) bool {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		log.Printf("warning: %s file not found: %s", label, path)
		r.GET(route, func(c *gin.Context) {
			c.String(404, "%s file not found: %s", label, path)
		})
		return false
	}
	if err != nil {
		log.Printf("warning: failed to stat %s: %v", label, err)
		r.GET(route, func(c *gin.Context) {
			c.String(500, "failed to stat %s: %v", label, err)
		})
		return false
	}

	log.Printf("found %s: %s (size: %d bytes)", label, path, fileInfo.Size())
	r.GET(route, func(c *gin.Context) { c.File(path) })
	log.Printf("%s route registered: http://localhost%s%s", label, serverAddress(), route)
	return true
}

func serverAddress() string {
	if config.Cfg != nil && config.Cfg.ServerConf.Address != "" {
		return config.Cfg.ServerConf.Address
	}
	return ":8081"
}
