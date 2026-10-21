package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-agent/rag/article"

	"github.com/gin-gonic/gin"
)

type fakeArticleIndexService struct {
	result article.IndexResult
	err    error
}

func (f *fakeArticleIndexService) Index(ctx context.Context, snapshot article.Snapshot) (article.IndexResult, error) {
	return f.result, f.err
}

func TestIndexArticleHandlerReturnsDualWriteResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/internal/v1/index/articles", strings.NewReader(`{
		"article_id": 12,
		"version": 2,
		"title": "Gin",
		"content": "Article body",
		"status": "published"
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	result := article.IndexResult{
		ChunkIDs: []string{"article:12:version:2:chunk:0"},
		ES:       article.BackendResult{IDs: []string{"article:12:version:2:chunk:0"}},
		Milvus:   article.BackendResult{IDs: []string{"article:12:version:2:chunk:0"}},
	}
	indexArticleHandler(&fakeArticleIndexService{result: result})(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Success    bool `json:"success"`
		ChunkCount int  `json:"chunk_count"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.ChunkCount != 1 {
		t.Fatalf("response = %+v", response)
	}
}

func TestIndexArticleHandlerRejectsDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/internal/v1/index/articles", strings.NewReader(`{
		"article_id": 12,
		"version": 2,
		"title": "Draft",
		"content": "Article body",
		"status": "draft"
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	indexArticleHandler(&fakeArticleIndexService{})(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
