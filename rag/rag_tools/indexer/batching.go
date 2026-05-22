package indexer

import (
	"context"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
)

const defaultStoreBatchSize = 10

type batchingIndexer struct {
	base      indexer.Indexer
	batchSize int
}

func newBatchingIndexer(base indexer.Indexer, batchSize int) indexer.Indexer {
	if batchSize <= 0 {
		batchSize = defaultStoreBatchSize
	}
	return &batchingIndexer{
		base:      base,
		batchSize: batchSize,
	}
}

func (b *batchingIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if len(docs) <= b.batchSize {
		return b.base.Store(ctx, docs, opts...)
	}

	ids := make([]string, 0, len(docs))
	for start := 0; start < len(docs); start += b.batchSize {
		end := start + b.batchSize
		if end > len(docs) {
			end = len(docs)
		}
		batchIDs, err := b.base.Store(ctx, docs[start:end], opts...)
		if err != nil {
			return ids, err
		}
		ids = append(ids, batchIDs...)
	}
	return ids, nil
}
