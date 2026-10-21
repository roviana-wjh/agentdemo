package article

import "testing"

func TestMilvusArticleDeleteExpressionTargetsEveryVersion(t *testing.T) {
	got := milvusArticleDeleteExpression(42)
	want := `metadata["article_id"] == 42`
	if got != want {
		t.Fatalf("expression = %q, want %q", got, want)
	}
}
