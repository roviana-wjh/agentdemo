package retriever

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-agent/config"
	"go-agent/rag/rag_tools/db"

	componentretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/elastic/go-elasticsearch/v8"
)

type esBM25Retriever struct {
	client *elasticsearch.Client
	index  string
	topK   int
}

func initES() {
	registerRetriever("es", func(ctx context.Context) (componentretriever.Retriever, error) {
		var err error
		if db.ES == nil {
			if db.ES, err = db.NewES(); err != nil {
				return nil, err
			}
		}
		return &esBM25Retriever{
			client: db.ES,
			index:  config.Cfg.ESConf.Index,
			topK:   config.Cfg.HybridSearchConf.ESTopK,
		}, nil
	})
}

func (r *esBM25Retriever) Retrieve(ctx context.Context, query string, opts ...componentretriever.Option) ([]*schema.Document, error) {
	options := componentretriever.GetCommonOptions(&componentretriever.Options{})
	options = componentretriever.GetCommonOptions(options, opts...)
	topK := r.topK
	if options.TopK != nil && *options.TopK > 0 {
		topK = *options.TopK
	}
	if topK <= 0 {
		topK = 20
	}

	requestBody := map[string]any{
		"size": topK,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"multi_match": map[string]any{
							"query":  query,
							"fields": []string{"title^3", "content", "tags^2"},
						},
					},
				},
				"filter": []any{
					map[string]any{"term": map[string]any{"status": "published"}},
				},
			},
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode ES BM25 query: %w", err)
	}
	response, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.index),
		r.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("execute ES BM25 query: %w", err)
	}
	defer response.Body.Close()
	if response.IsError() {
		return nil, fmt.Errorf("ES BM25 query returned %s", response.Status())
	}

	var payload struct {
		Hits struct {
			Hits []struct {
				ID     string         `json:"_id"`
				Score  float64        `json:"_score"`
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode ES BM25 response: %w", err)
	}

	docs := make([]*schema.Document, 0, len(payload.Hits.Hits))
	for _, hit := range payload.Hits.Hits {
		id := hit.ID
		if chunkID, ok := hit.Source["chunk_id"].(string); ok && chunkID != "" {
			id = chunkID
		}
		content, _ := hit.Source["content"].(string)
		metadata := make(map[string]any)
		if rawMetadata, ok := hit.Source["metadata"].(map[string]any); ok {
			for key, value := range rawMetadata {
				metadata[key] = value
			}
		}
		for _, key := range []string{"article_id", "title", "tags", "status", "version", "published_at", "url"} {
			if value, ok := hit.Source[key]; ok {
				metadata[key] = value
			}
		}
		metadata["retrieval_source"] = "elasticsearch"
		metadata["bm25_score"] = hit.Score
		doc := &schema.Document{ID: id, Content: content, MetaData: metadata}
		doc.WithScore(hit.Score)
		docs = append(docs, doc)
	}
	return docs, nil
}
