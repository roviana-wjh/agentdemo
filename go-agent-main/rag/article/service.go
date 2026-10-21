package article

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-agent/config"
	"go-agent/rag/rag_tools/db"
	ragindexer "go-agent/rag/rag_tools/indexer"
	"net/http"
	"sync"

	"github.com/cloudwego/eino/components/document"
	componentindexer "github.com/cloudwego/eino/components/indexer"
)

type IndexerFactory func(ctx context.Context) (componentindexer.Indexer, error)

type IndexService struct {
	splitter      document.Transformer
	esFactory     IndexerFactory
	milvusFactory IndexerFactory
	mu            sync.Mutex
	esIndexer     componentindexer.Indexer
	milvusIndexer componentindexer.Indexer
}

var (
	defaultServiceMu sync.RWMutex
	defaultService   *IndexService
)

func NewDefaultIndexService(splitter document.Transformer) *IndexService {
	return NewIndexService(
		splitter,
		func(ctx context.Context) (componentindexer.Indexer, error) {
			if db.ES == nil {
				client, err := db.NewES()
				if err != nil {
					return nil, err
				}
				db.ES = client
			}
			return NewElasticsearchIndexer(db.ES, config.Cfg.ESConf.Index), nil
		},
		func(ctx context.Context) (componentindexer.Indexer, error) {
			if db.Milvus == nil {
				return nil, fmt.Errorf("Milvus client is not initialized")
			}
			return ragindexer.GetIndexer(ctx, "milvus")
		},
	)
}

func NewIndexService(splitter document.Transformer, esFactory, milvusFactory IndexerFactory) *IndexService {
	return &IndexService{
		splitter:      splitter,
		esFactory:     esFactory,
		milvusFactory: milvusFactory,
	}
}

func SetDefaultIndexService(service *IndexService) {
	defaultServiceMu.Lock()
	defer defaultServiceMu.Unlock()
	defaultService = service
}

func GetDefaultIndexService() *IndexService {
	defaultServiceMu.RLock()
	defer defaultServiceMu.RUnlock()
	return defaultService
}

func (s *IndexService) Index(ctx context.Context, snapshot Snapshot) (IndexResult, error) {
	docs, err := BuildDocuments(ctx, snapshot, s.splitter)
	if err != nil {
		return IndexResult{}, err
	}
	result := IndexResult{ChunkIDs: make([]string, 0, len(docs))}
	for _, doc := range docs {
		result.ChunkIDs = append(result.ChunkIDs, doc.ID)
	}

	type outcome struct {
		backend string
		ids     []string
		err     error
	}
	outcomes := make(chan outcome, 2)
	for _, backend := range []string{"elasticsearch", "milvus"} {
		backend := backend
		go func() {
			indexer, indexerErr := s.getIndexer(ctx, backend)
			if indexerErr != nil {
				outcomes <- outcome{backend: backend, err: indexerErr}
				return
			}
			if cleanupErr := deleteExistingArticleChunks(ctx, backend, snapshot.ArticleID); cleanupErr != nil {
				outcomes <- outcome{backend: backend, err: cleanupErr}
				return
			}
			ids, storeErr := indexer.Store(ctx, docs)
			outcomes <- outcome{backend: backend, ids: ids, err: storeErr}
		}()
	}

	failed := false
	for count := 0; count < 2; count++ {
		outcome := <-outcomes
		backendResult := BackendResult{IDs: outcome.ids}
		if outcome.err != nil {
			failed = true
			backendResult.Error = outcome.err.Error()
		}
		if outcome.backend == "elasticsearch" {
			result.ES = backendResult
		} else {
			result.Milvus = backendResult
		}
	}
	if failed {
		return result, &IndexError{Result: result}
	}
	return result, nil
}

// deleteExistingArticleChunks makes indexing replace-by-article rather than
// append-by-version. A retry of the same snapshot and a newer version both
// converge to exactly one searchable article version per backend.
func deleteExistingArticleChunks(ctx context.Context, backend string, articleID int64) error {
	if articleID <= 0 {
		return fmt.Errorf("invalid article id %d", articleID)
	}
	switch backend {
	case "elasticsearch":
		if db.ES == nil {
			return nil
		}
		body, err := json.Marshal(map[string]any{
			"query": map[string]any{"term": map[string]any{"article_id": articleID}},
		})
		if err != nil {
			return err
		}
		response, err := db.ES.DeleteByQuery(
			[]string{config.Cfg.ESConf.Index},
			bytes.NewReader(body),
			db.ES.DeleteByQuery.WithContext(ctx),
			db.ES.DeleteByQuery.WithRefresh(true),
			db.ES.DeleteByQuery.WithConflicts("proceed"),
		)
		if err != nil {
			return fmt.Errorf("delete stale ES chunks for article %d: %w", articleID, err)
		}
		defer response.Body.Close()
		if response.IsError() && response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("delete stale ES chunks for article %d returned %s", articleID, response.Status())
		}
		return nil
	case "milvus":
		if db.Milvus == nil {
			return nil
		}
		expression := milvusArticleDeleteExpression(articleID)
		if err := db.Milvus.Delete(ctx, config.Cfg.MilvusConf.CollectionName, "", expression); err != nil {
			return fmt.Errorf("delete stale Milvus chunks for article %d: %w", articleID, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported index backend %q", backend)
	}
}

func milvusArticleDeleteExpression(articleID int64) string {
	return fmt.Sprintf(`metadata["article_id"] == %d`, articleID)
}

func (s *IndexService) getIndexer(ctx context.Context, backend string) (componentindexer.Indexer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if backend == "elasticsearch" {
		if s.esIndexer == nil {
			indexer, err := s.esFactory(ctx)
			if err != nil {
				return nil, err
			}
			s.esIndexer = indexer
		}
		return s.esIndexer, nil
	}
	if s.milvusIndexer == nil {
		indexer, err := s.milvusFactory(ctx)
		if err != nil {
			return nil, err
		}
		s.milvusIndexer = indexer
	}
	return s.milvusIndexer, nil
}
