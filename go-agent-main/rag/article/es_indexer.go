package article

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	componentindexer "github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8"
)

type ElasticsearchIndexer struct {
	client *elasticsearch.Client
	index  string
}

func NewElasticsearchIndexer(client *elasticsearch.Client, index string) *ElasticsearchIndexer {
	return &ElasticsearchIndexer{client: client, index: index}
}

func (i *ElasticsearchIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...componentindexer.Option) ([]string, error) {
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		source := map[string]any{
			"chunk_id": doc.ID,
			"content":  doc.Content,
			"metadata": doc.MetaData,
		}
		for _, key := range []string{"article_id", "title", "tags", "status", "version", "published_at", "url"} {
			if value, ok := doc.MetaData[key]; ok {
				source[key] = value
			}
		}
		body, err := json.Marshal(source)
		if err != nil {
			return ids, fmt.Errorf("encode ES document %s: %w", doc.ID, err)
		}
		response, err := i.client.Index(
			i.index,
			bytes.NewReader(body),
			i.client.Index.WithContext(ctx),
			i.client.Index.WithDocumentID(doc.ID),
		)
		if err != nil {
			return ids, fmt.Errorf("index ES document %s: %w", doc.ID, err)
		}
		response.Body.Close()
		if response.IsError() {
			return ids, fmt.Errorf("index ES document %s returned %s", doc.ID, response.Status())
		}
		ids = append(ids, doc.ID)
	}
	return ids, nil
}
