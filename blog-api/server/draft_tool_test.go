package server

import (
	"awesomeProject1/model"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeDraftToolRepository struct {
	created       *model.Article
	updated       model.Article
	updateErr     error
	updateAuthor  uint
	updateArticle uint
}

func (f *fakeDraftToolRepository) CreateDraft(_ context.Context, article *model.Article) error {
	copy := *article
	copy.ID = 12
	*article = copy
	f.created = &copy
	return nil
}

func (f *fakeDraftToolRepository) UpdateOwnDraft(_ context.Context, authorID, articleID uint, title, content, image string) (model.Article, error) {
	f.updateAuthor, f.updateArticle = authorID, articleID
	if f.updateErr != nil {
		return model.Article{}, f.updateErr
	}
	return f.updated, nil
}

func withFakeDraftTools(t *testing.T, fake draftToolRepository) {
	t.Helper()
	original := draftToolsRepository
	draftToolsRepository = fake
	t.Cleanup(func() { draftToolsRepository = original })
}

func performDraftRequest(handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	router := gin.New()
	router.POST("/draft", handler)
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/draft", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, req)
	return response
}

func TestCreateArticleDraftToolCreatesForcedDraftAndReturnsSafeDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakeDraftToolRepository{}
	withFakeDraftTools(t, fake)

	response := performDraftRequest(CreateArticleDraftTool, `{"author_id":7,"title":"  New post  ","content":"body","image":"cover.png","status":"published"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if fake.created == nil || fake.created.AuthorID != 7 || fake.created.Status != "draft" || fake.created.Title != "New post" || fake.created.Version != 1 {
		t.Fatalf("unexpected created article: %+v", fake.created)
	}
	var payload struct {
		Data articleDraftResult `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.ID != 12 || payload.Data.Status != "draft" || payload.Data.Title != "New post" {
		t.Fatalf("unexpected response: %+v", payload.Data)
	}
	for _, forbidden := range []string{"author_id", "content", "image", "password", "published"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("response exposed forbidden field/value %q: %s", forbidden, response.Body.String())
		}
	}
}

func TestUpdateArticleDraftToolRejectsOtherAuthorOrNonDraft(t *testing.T) {
	fake := &fakeDraftToolRepository{updateErr: errDraftNotFound}
	withFakeDraftTools(t, fake)

	response := performDraftRequest(UpdateArticleDraftTool, `{"author_id":7,"article_id":9,"title":"changed","content":"body"}`)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
	if fake.updateAuthor != 7 || fake.updateArticle != 9 {
		t.Fatalf("ownership selector = author %d article %d", fake.updateAuthor, fake.updateArticle)
	}

	fake.updateErr = nil
	fake.updated = model.Article{Status: "published"}
	response = performDraftRequest(UpdateArticleDraftTool, `{"author_id":7,"article_id":9,"title":"changed","content":"body"}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestDraftToolsValidateRequiredFields(t *testing.T) {
	fake := &fakeDraftToolRepository{}
	withFakeDraftTools(t, fake)

	tests := []struct {
		name    string
		handler gin.HandlerFunc
		body    string
	}{
		{name: "create missing author", handler: CreateArticleDraftTool, body: `{"title":"title","content":"body"}`},
		{name: "create blank title", handler: CreateArticleDraftTool, body: `{"author_id":1,"title":"  ","content":"body"}`},
		{name: "create blank content", handler: CreateArticleDraftTool, body: `{"author_id":1,"title":"title","content":"  "}`},
		{name: "update missing article", handler: UpdateArticleDraftTool, body: `{"author_id":1,"title":"title","content":"body"}`},
		{name: "malformed json", handler: CreateArticleDraftTool, body: `{"author_id":`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performDraftRequest(test.handler, test.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
	if fake.created != nil || fake.updateAuthor != 0 {
		t.Fatal("repository called for invalid request")
	}
}

func TestUpdateArticleDraftToolHandlesRepositoryFailure(t *testing.T) {
	fake := &fakeDraftToolRepository{updateErr: errors.New("database unavailable")}
	withFakeDraftTools(t, fake)
	response := performDraftRequest(UpdateArticleDraftTool, `{"author_id":1,"article_id":2,"title":"title","content":"body"}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
