package server

import (
	"awesomeProject1/model"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestArticleTagsDatabaseAndJSONRoundTrip(t *testing.T) {
	original := model.Tags{"Go", "RAG"}
	value, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var decoded model.Tags
	if err := decoded.Scan(value); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[0] != "Go" || decoded[1] != "RAG" {
		t.Fatalf("decoded tags = %#v", decoded)
	}
	emptyJSON, err := json.Marshal(model.Tags(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(emptyJSON) != "[]" {
		t.Fatalf("nil tags JSON = %s, want []", emptyJSON)
	}
}

type fakeHotTagsRepository struct {
	tags []model.Tags
}

func (f fakeHotTagsRepository) ListPublishedArticleTags(context.Context) ([]model.Tags, error) {
	return f.tags, nil
}

type fakeBlogStatisticsRepository struct {
	result blogStatistics
}

func (f fakeBlogStatisticsRepository) GetPublishedBlogStatistics(context.Context) (blogStatistics, error) {
	return f.result, nil
}

func TestGetHotTagsToolNormalizesRanksAndLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	original := hotTagsRepo
	hotTagsRepo = fakeHotTagsRepository{tags: []model.Tags{
		{" Go ", "RAG", "go"},
		{"Go", "Vue"},
		{"RAG"},
	}}
	t.Cleanup(func() { hotTagsRepo = original })

	router := gin.New()
	router.GET("/tags", GetHotTagsTool)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/tags?limit=2", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data []hotTag `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 2 || payload.Data[0].Tag != "Go" || payload.Data[0].Count != 2 || payload.Data[1].Tag != "RAG" || payload.Data[1].Count != 2 {
		t.Fatalf("unexpected payload: %+v", payload.Data)
	}
}

func TestGetBlogStatisticsToolReturnsSafeContract(t *testing.T) {
	original := blogStatisticsRepo
	blogStatisticsRepo = fakeBlogStatisticsRepository{result: blogStatistics{ArticleCount: 4, TotalViews: 120, CommentCount: 9}}
	t.Cleanup(func() { blogStatisticsRepo = original })

	router := gin.New()
	router.GET("/statistics", GetBlogStatisticsTool)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/statistics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Body.String() != `{"data":{"article_count":4,"total_views":120,"comment_count":9}}` {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestNormalizeTagsRemovesBlanksAndCaseInsensitiveDuplicates(t *testing.T) {
	got := normalizeTags([]string{" Go ", "", "go", "RAG"})
	if len(got) != 2 || got[0] != "Go" || got[1] != "RAG" {
		t.Fatalf("normalizeTags() = %#v", got)
	}
}
