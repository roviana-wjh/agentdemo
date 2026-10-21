package blog_tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientReadsTask12AnalyticsContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/internal/v1/tools/get_hot_tags":
			_, _ = w.Write([]byte(`{"data":[{"tag":"Go","count":2}]}`))
		case "/internal/v1/tools/get_blog_statistics":
			_, _ = w.Write([]byte(`{"data":{"article_count":4,"total_views":120,"comment_count":9}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	tags, err := client.GetHotTags(context.Background(), 10)
	if err != nil || len(tags.Tags) != 1 || tags.Tags[0].Tag != "Go" || tags.Tags[0].Count != 2 {
		t.Fatalf("tags = %+v, err = %v", tags, err)
	}
	stats, err := client.GetBlogStatistics(context.Background())
	if err != nil || stats.ArticleCount != 4 || stats.TotalViews != 120 || stats.CommentCount != 9 {
		t.Fatalf("stats = %+v, err = %v", stats, err)
	}
}
