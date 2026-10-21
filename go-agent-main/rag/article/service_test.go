package article

import (
	"context"
	"errors"
	"testing"

	"go-agent/tool/document"

	componentindexer "github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
)

type fakeIndexer struct {
	ids []string
	err error
}

func (f *fakeIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...componentindexer.Option) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.ids != nil {
		return f.ids, nil
	}
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}
	return ids, nil
}

func TestBuildDocumentsUsesStableChunkIDsAndMetadata(t *testing.T) {
	splitter, err := document.NewSplitter(context.Background())
	if err != nil {
		t.Fatalf("NewSplitter() error = %v", err)
	}
	docs, err := BuildDocuments(context.Background(), Snapshot{
		ArticleID: 12,
		Version:   3,
		Title:     "Gin Middleware",
		Content:   "# Intro\n\nGin is **fast**. [Docs](https://gin-gonic.com).",
		Tags:      []string{"Go", "Gin"},
		Status:    "published",
	}, splitter)
	if err != nil {
		t.Fatalf("BuildDocuments() error = %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("BuildDocuments() returned no chunks")
	}
	if got := docs[0].ID; got != "article:12:version:3:chunk:0" {
		t.Fatalf("chunk ID = %q", got)
	}
	if got := docs[0].MetaData["article_id"]; got != int64(12) {
		t.Fatalf("article_id metadata = %#v", got)
	}
	if docs[0].Content == "" || docs[0].Content == "# Intro" {
		t.Fatalf("cleaned content = %q", docs[0].Content)
	}
}

func TestIndexServiceWritesBothBackends(t *testing.T) {
	splitter, err := document.NewSplitter(context.Background())
	if err != nil {
		t.Fatalf("NewSplitter() error = %v", err)
	}
	service := NewIndexService(
		splitter,
		func(context.Context) (componentindexer.Indexer, error) { return &fakeIndexer{}, nil },
		func(context.Context) (componentindexer.Indexer, error) { return &fakeIndexer{}, nil },
	)
	result, err := service.Index(context.Background(), Snapshot{
		ArticleID: 1,
		Version:   1,
		Title:     "Article",
		Content:   "Article body",
		Status:    "published",
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}
	if len(result.ChunkIDs) == 0 || len(result.ES.IDs) != len(result.ChunkIDs) || len(result.Milvus.IDs) != len(result.ChunkIDs) {
		t.Fatalf("unexpected result = %+v", result)
	}
}

func TestIndexServiceReturnsPartialBackendFailure(t *testing.T) {
	splitter, err := document.NewSplitter(context.Background())
	if err != nil {
		t.Fatalf("NewSplitter() error = %v", err)
	}
	service := NewIndexService(
		splitter,
		func(context.Context) (componentindexer.Indexer, error) { return &fakeIndexer{}, nil },
		func(context.Context) (componentindexer.Indexer, error) {
			return &fakeIndexer{err: errors.New("milvus unavailable")}, nil
		},
	)
	result, err := service.Index(context.Background(), Snapshot{
		ArticleID: 1,
		Version:   1,
		Title:     "Article",
		Content:   "Article body",
		Status:    "published",
	})
	var indexErr *IndexError
	if !errors.As(err, &indexErr) {
		t.Fatalf("Index() error = %v, want IndexError", err)
	}
	if len(result.ES.IDs) == 0 || result.Milvus.Error == "" {
		t.Fatalf("unexpected partial result = %+v", result)
	}
}

func TestSnapshotRejectsDraft(t *testing.T) {
	err := (Snapshot{ArticleID: 1, Version: 1, Title: "Draft", Content: "body", Status: "draft"}).Validate()
	if err == nil {
		t.Fatal("Validate() accepted a draft")
	}
}
