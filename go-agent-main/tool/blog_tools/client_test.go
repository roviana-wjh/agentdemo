package blog_tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUsesAuthenticatedInternalDomainEndpoints(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Internal-Token"); got != "test-secret" {
			t.Errorf("X-Internal-Token = %q", got)
		}
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/internal/v1/tools/search_article", "/internal/v1/tools/get_hot_articles":
			_, _ = w.Write([]byte(`{"data":[{"id":12,"title":"Gin","author_id":7,"author_name":"Ada"}]}`))
		case "/internal/v1/tools/get_article_detail":
			_, _ = w.Write([]byte(`{"data":{"id":12,"title":"Gin"}}`))
		case "/internal/v1/tools/get_recent_comments":
			_, _ = w.Write([]byte(`{"data":[{"id":3,"article_id":12,"content":"nice","user_id":8,"user_name":"Lin"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-secret", server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	ctx := context.Background()
	if result, err := client.SearchArticle(ctx, "Gin middleware", 5); err != nil || len(result.Articles) != 1 {
		t.Fatalf("SearchArticle() = %+v, %v", result, err)
	} else if result.Articles[0].AuthorName != "Ada" {
		t.Fatalf("SearchArticle() author_name = %q", result.Articles[0].AuthorName)
	}
	if result, err := client.GetArticleDetail(ctx, 12); err != nil || result.Article.ID != 12 {
		t.Fatalf("GetArticleDetail() = %+v, %v", result, err)
	}
	if result, err := client.GetHotArticles(ctx, 3); err != nil || len(result.Articles) != 1 {
		t.Fatalf("GetHotArticles() = %+v, %v", result, err)
	}
	if result, err := client.GetRecentComments(ctx, 12, 8); err != nil || len(result.Comments) != 1 {
		t.Fatalf("GetRecentComments() = %+v, %v", result, err)
	} else if result.Comments[0].UserName != "Lin" {
		t.Fatalf("GetRecentComments() user_name = %q", result.Comments[0].UserName)
	}

	wants := []string{
		"/internal/v1/tools/search_article?limit=5&q=Gin+middleware",
		"/internal/v1/tools/get_article_detail?id=12",
		"/internal/v1/tools/get_hot_articles?limit=3",
		"/internal/v1/tools/get_recent_comments?article_id=12&limit=8",
	}
	for i := range wants {
		if requests[i] != wants[i] {
			t.Errorf("request[%d] = %q, want %q", i, requests[i], wants[i])
		}
	}
}

func TestClientReportsUpstreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "wrong-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetHotArticles(context.Background(), 5); err == nil {
		t.Fatal("GetHotArticles() error = nil")
	}
}
