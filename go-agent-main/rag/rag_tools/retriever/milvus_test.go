package retriever

import (
	"context"
	"testing"

	milvusretriever "github.com/cloudwego/eino-ext/components/retriever/milvus"
	componentretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type capturingRetriever struct {
	filter string
}

func (r *capturingRetriever) Retrieve(_ context.Context, _ string, opts ...componentretriever.Option) ([]*schema.Document, error) {
	impl := componentretriever.GetImplSpecificOptions(&milvusretriever.ImplOptions{}, opts...)
	r.filter = impl.Filter
	return []*schema.Document{}, nil
}

func TestPublishedMilvusRetrieverAlwaysAppliesPublishedFilter(t *testing.T) {
	base := &capturingRetriever{}
	retriever := &publishedMilvusRetriever{base: base}
	if _, err := retriever.Retrieve(context.Background(), "query"); err != nil {
		t.Fatal(err)
	}
	if base.filter != `metadata["status"] == "published"` {
		t.Fatalf("filter = %q", base.filter)
	}
}
