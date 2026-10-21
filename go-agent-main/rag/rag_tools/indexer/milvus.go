package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"go-agent/config"
	"go-agent/model/embedding_model"
	"go-agent/rag/rag_tools/db"
	"log"
	"strconv"

	"github.com/cloudwego/eino-ext/components/indexer/milvus"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type floatSchema struct {
	ID       string    `json:"id" milvus:"name:id"`
	Content  string    `json:"content" milvus:"name:content"`
	Vector   []float32 `json:"vector" milvus:"name:vector"`
	Metadata []byte    `json:"metadata" milvus:"name:metadata"`
}

func initMilvus() {
	registerIndexer("milvus", func(ctx context.Context) (indexer.Indexer, error) {
		dim, err := getEmbeddingDim(ctx)
		if err != nil {
			return nil, err
		}
		log.Printf("embedding dim: %d", dim)

		if err := validateCollectionSchema(ctx, config.Cfg.MilvusConf.CollectionName, dim); err != nil {
			return nil, err
		}

		idx, err := milvus.NewIndexer(ctx, buildMilvusIndexerConfig(dim))
		if err != nil {
			return nil, fmt.Errorf("create Milvus indexer without modifying existing collection: %w", err)
		}

		return newBatchingIndexer(idx, defaultStoreBatchSize), nil
	})
}

func buildMilvusIndexerConfig(dim int) *milvus.IndexerConfig {
	emb, err := embedding_model.GetEmbeddingModel(context.Background(), config.Cfg.EmbeddingModelType)
	if err != nil {
		return nil
	}

	return &milvus.IndexerConfig{
		Client:     db.Milvus,
		Embedding:  emb,
		Collection: config.Cfg.MilvusConf.CollectionName,
		MetricType: milvus.COSINE,
		Fields: []*entity.Field{
			entity.NewField().WithName("id").WithDescription("document id").WithIsPrimaryKey(true).WithDataType(entity.FieldTypeVarChar).WithMaxLength(255),
			entity.NewField().WithName("vector").WithDescription("document vector").WithIsPrimaryKey(false).WithDataType(entity.FieldTypeFloatVector).WithDim(int64(dim)),
			entity.NewField().WithName("content").WithDescription("document content").WithIsPrimaryKey(false).WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535),
			entity.NewField().WithName("metadata").WithDescription("document metadata").WithIsPrimaryKey(false).WithDataType(entity.FieldTypeJSON),
		},
		DocumentConverter: func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]interface{}, error) {
			if len(vectors) != len(docs) {
				return nil, fmt.Errorf("vector size mismatch, docs=%d vectors=%d", len(docs), len(vectors))
			}

			rows := make([]interface{}, 0, len(docs))
			for i, doc := range docs {
				metadata, err := json.Marshal(doc.MetaData)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal metadata: %w", err)
				}
				vec := make([]float32, len(vectors[i]))
				for j, v := range vectors[i] {
					vec[j] = float32(v)
				}
				rows = append(rows, &floatSchema{
					ID:       doc.ID,
					Content:  doc.Content,
					Vector:   vec,
					Metadata: metadata,
				})
			}

			return rows, nil
		},
	}
}

func getEmbeddingDim(ctx context.Context) (int, error) {
	emb, err := embedding_model.GetEmbeddingModel(context.Background(), config.Cfg.EmbeddingModelType)
	if err != nil {
		return 0, err
	}

	vecs, err := emb.EmbedStrings(ctx, []string{"dim"})
	if err != nil {
		return 0, fmt.Errorf("failed to get embedding dim: %w", err)
	}
	if len(vecs) != 1 || len(vecs[0]) == 0 {
		return 0, fmt.Errorf("invalid embedding dim result")
	}
	return len(vecs[0]), nil
}

func validateCollectionSchema(ctx context.Context, collectionName string, expectedDim int) error {
	exists, err := db.Milvus.HasCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("check collection exists failed: %w", err)
	}
	if !exists {
		return nil
	}

	coll, err := db.Milvus.DescribeCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("describe collection failed: %w", err)
	}

	fields := map[string]entity.FieldType{}
	vectorDim := 0
	for _, field := range coll.Schema.Fields {
		fields[field.Name] = field.DataType
		if field.Name == "vector" && field.DataType == entity.FieldTypeFloatVector {
			dimStr := field.TypeParams["dim"]
			if dim, err := strconv.Atoi(dimStr); err == nil {
				vectorDim = dim
			}
		}
	}

	schemaMismatch := fields["id"] != entity.FieldTypeVarChar ||
		fields["vector"] != entity.FieldTypeFloatVector ||
		fields["content"] != entity.FieldTypeVarChar ||
		fields["metadata"] != entity.FieldTypeJSON ||
		vectorDim != expectedDim

	if !schemaMismatch {
		return nil
	}

	return fmt.Errorf(
		"Milvus collection %s schema mismatch: expected vector dimension %d; choose a new collection name or rebuild it manually",
		collectionName,
		expectedDim,
	)
}
