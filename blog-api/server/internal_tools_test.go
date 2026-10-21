package server

import (
	"awesomeProject1/middleware"
	"awesomeProject1/model"
	"awesomeProject1/setting"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestInternalToolAuthFailsClosedAndAcceptsConfiguredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := setting.InternalToolSetting.Token
	t.Cleanup(func() { setting.InternalToolSetting.Token = original })
	router := gin.New()
	router.GET("/internal", middleware.InternalAuthMiddleware, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, test := range []struct {
		name       string
		configured string
		provided   string
		wantStatus int
	}{
		{name: "empty configuration", wantStatus: http.StatusUnauthorized},
		{name: "missing header", configured: "secret", wantStatus: http.StatusUnauthorized},
		{name: "wrong header", configured: "secret", provided: "wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid header", configured: "secret", provided: "secret", wantStatus: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			setting.InternalToolSetting.Token = test.configured
			req := httptest.NewRequest(http.MethodGet, "/internal", nil)
			if test.provided != "" {
				req.Header.Set(middleware.InternalTokenHeader, test.provided)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

type fakeInternalToolRepository struct {
	articles []internalArticleRecord
	article  internalArticleRecord
	comments []internalCommentRecord
	query    string
	limit    int
}

func (f *fakeInternalToolRepository) SearchPublishedArticles(_ context.Context, query string, limit int) ([]internalArticleRecord, error) {
	f.query, f.limit = query, limit
	return f.articles, nil
}

func (f *fakeInternalToolRepository) GetPublishedArticle(_ context.Context, _ uint) (internalArticleRecord, error) {
	return f.article, nil
}

func (f *fakeInternalToolRepository) GetHotPublishedArticles(_ context.Context, limit int) ([]internalArticleRecord, error) {
	f.limit = limit
	return f.articles, nil
}

func (f *fakeInternalToolRepository) GetRecentPublishedComments(_ context.Context, _ uint, limit int) ([]internalCommentRecord, error) {
	f.limit = limit
	return f.comments, nil
}

func withFakeInternalTools(t *testing.T, fake internalToolRepository) {
	t.Helper()
	original := internalToolsRepository
	internalToolsRepository = fake
	t.Cleanup(func() { internalToolsRepository = original })
}

func TestSearchArticleToolFiltersDraftsAndDoesNotExposeCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeInternalToolRepository{articles: []internalArticleRecord{
		{Article: model.Article{Model: modelModel(1), Title: "Published", Status: "published", AuthorID: 7, Author: model.User{Model: modelModel(7), Name: "alice", Password: "hash"}}, CommentCount: 4},
		{Article: model.Article{Model: modelModel(2), Title: "Draft", Status: "draft", Author: model.User{Password: "do-not-leak"}}},
	}}
	withFakeInternalTools(t, fake)

	router := gin.New()
	router.GET("/search", SearchArticleTool)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/search?q=gin&limit=100", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if fake.query != "gin" || fake.limit != maxToolLimit {
		t.Fatalf("repository args = %q, %d", fake.query, fake.limit)
	}
	body := response.Body.String()
	if strings.Contains(body, "Draft") || strings.Contains(body, "hash") || strings.Contains(body, "password") {
		t.Fatalf("response exposed draft or credential: %s", body)
	}
	var payload struct {
		Data []internalArticle `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 1 || payload.Data[0].AuthorName != "alice" || payload.Data[0].CommentCount != 4 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestGetRecentCommentsToolReturnsSafeUserShape(t *testing.T) {
	now := time.Now()
	fake := &fakeInternalToolRepository{comments: []internalCommentRecord{{
		Comment: model.Comment{ID: 3, ArticleID: 8, UserID: 4, Content: "nice", CreatedAt: now,
			User: model.User{Model: modelModel(4), Name: "bob", Password: "secret"}},
		ArticleTitle: "Published article",
	}}}
	withFakeInternalTools(t, fake)
	router := gin.New()
	router.GET("/comments", GetRecentCommentsTool)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/comments?limit=2", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "secret") || strings.Contains(body, "password") {
		t.Fatalf("response exposed credentials: %s", body)
	}
	if !strings.Contains(body, `"user_name":"bob"`) || !strings.Contains(body, `"article_title":"Published article"`) {
		t.Fatalf("response omitted safe user name: %s", body)
	}
}

// modelModel keeps test fixtures readable without relying on implicit IDs.
func modelModel(id uint) gorm.Model {
	return gorm.Model{ID: id}
}
