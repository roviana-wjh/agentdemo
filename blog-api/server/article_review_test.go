package server

import (
	"awesomeProject1/model"
	"testing"
)

func TestArticleReviewIsCurrent(t *testing.T) {
	article := model.Article{
		Title: "Go Context", Content: "content", Image: "cover.png", Tags: model.Tags{"Go"},
		ReviewStatus: reviewStatusApproved,
	}
	article.ReviewHash = hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
	if !articleReviewIsCurrent(article) {
		t.Fatal("approved matching snapshot should be publishable")
	}

	article.Content = "changed"
	if articleReviewIsCurrent(article) {
		t.Fatal("changed content must invalidate the review")
	}
}

func TestArticleReviewRequiresApprovedStatus(t *testing.T) {
	article := model.Article{Title: "title", Content: "content", ReviewStatus: "needs_revision"}
	article.ReviewHash = hashArticleContentWithTags(article.Title, article.Content, article.Image, article.Tags)
	if articleReviewIsCurrent(article) {
		t.Fatal("non-approved review must not pass the publish gate")
	}
}
