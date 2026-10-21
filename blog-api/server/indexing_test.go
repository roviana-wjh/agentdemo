package server

import (
	"awesomeProject1/model"
	"testing"
	"time"
)

func TestHashArticleContentIsStableAndContentSensitive(t *testing.T) {
	first := hashArticleContent("title", "content", "image")
	second := hashArticleContent("title", "content", "image")
	changed := hashArticleContent("title", "changed", "image")
	if first != second {
		t.Fatalf("stable input produced different hashes")
	}
	if first == changed {
		t.Fatalf("changed content produced the same hash")
	}
	if len(first) != 64 {
		t.Fatalf("hash length = %d", len(first))
	}
	withoutTags := hashArticleContentWithTags("title", "content", "image", nil)
	withTags := hashArticleContentWithTags("title", "content", "image", model.Tags{"Go"})
	if withoutTags == withTags {
		t.Fatal("changed tags produced the same hash")
	}
}

func TestBuildArticleIndexRequest(t *testing.T) {
	publishedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	article := model.Article{
		Title:       "Gin",
		Content:     "Article body",
		Tags:        model.Tags{"Go", "Gin"},
		Status:      "published",
		Version:     2,
		PublishedAt: &publishedAt,
	}
	article.ID = 9

	request := buildArticleIndexRequest(article)
	if request.ArticleID != 9 || request.Version != 2 || request.URL != "/article/9" {
		t.Fatalf("request = %+v", request)
	}
	if len(request.Tags) != 2 || request.Tags[0] != "Go" || request.Tags[1] != "Gin" {
		t.Fatalf("tags = %#v", request.Tags)
	}
	if request.PublishedAt == nil || !request.PublishedAt.Equal(publishedAt) {
		t.Fatalf("published_at = %v", request.PublishedAt)
	}
}
