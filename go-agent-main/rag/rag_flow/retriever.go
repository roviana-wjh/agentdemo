package rag_flow

import (
	"context"
	"go-agent/algorithm"
	"go-agent/config"
	"go-agent/model/chat_model"
	"go-agent/rag/rag_tools/reranker"
	toolretriever "go-agent/rag/rag_tools/retriever"
	"go-agent/tool"
	"log"
	"sync"

	componentretriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	MilvusRetriever = "MilvusRetriever"
	ESRetriever     = "ESRetriever"
	Fusion          = "RRFFusion"
	Reranker        = "Reranker"
	Trans_String    = "Trans_String"
)

type RetrievalState struct {
	Query string
}

type lazySafeRetriever struct {
	name string
	mu   sync.Mutex
	base componentretriever.Retriever
}

func (r *lazySafeRetriever) Retrieve(ctx context.Context, query string, opts ...componentretriever.Option) ([]*schema.Document, error) {
	r.mu.Lock()
	if r.base == nil {
		base, err := toolretriever.GetRetriever(ctx, r.name)
		if err != nil {
			r.mu.Unlock()
			log.Printf("warning: %s retriever initialization failed: %v", r.name, err)
			return []*schema.Document{}, nil
		}
		r.base = base
	}
	base := r.base
	r.mu.Unlock()

	docs, err := base.Retrieve(ctx, query, opts...)
	if err != nil {
		log.Printf("warning: %s retrieval failed: %v", r.name, err)
		return []*schema.Document{}, nil
	}
	return docs, nil
}

func BuildRetrieverGraph(ctx context.Context) (*compose.Graph[[]*schema.Message, []*schema.Document], error) {
	searchConfig := config.Cfg.HybridSearchConf
	localReranker := reranker.NewLexicalReranker()
	var resultReranker reranker.Reranker = localReranker
	if searchConfig.RerankMode == "llm" {
		if chat, err := chat_model.GetChatModel(ctx, config.Cfg.ChatModelType); err == nil {
			resultReranker = reranker.NewLLMReranker(chat, localReranker, searchConfig.RerankMaxContentLength)
		} else {
			log.Printf("warning: LLM reranker initialization failed, using lexical fallback: %v", err)
		}
	}

	g := compose.NewGraph[[]*schema.Message, []*schema.Document](
		compose.WithGenLocalState(func(ctx context.Context) *RetrievalState {
			return &RetrievalState{}
		}),
	)
	_ = g.AddLambdaNode(Trans_String, compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (string, error) {
		query, err := tool.MsgsToQuery(ctx, input)
		if err != nil {
			return "", err
		}
		err = compose.ProcessState[*RetrievalState](ctx, func(ctx context.Context, state *RetrievalState) error {
			state.Query = query
			return nil
		})
		return query, err
	}))

	_ = g.AddRetrieverNode(ESRetriever, &lazySafeRetriever{name: "es"}, compose.WithOutputKey("es"))
	_ = g.AddRetrieverNode(MilvusRetriever, &lazySafeRetriever{name: "milvus"}, compose.WithOutputKey("milvus"))
	_ = g.AddLambdaNode(Fusion, compose.InvokableLambda(func(ctx context.Context, input map[string]any) ([]*schema.Document, error) {
		esDocs, _ := input["es"].([]*schema.Document)
		milvusDocs, _ := input["milvus"].([]*schema.Document)
		results, err := algorithm.RRFFusionWithK(ctx, [][]*schema.Document{esDocs, milvusDocs}, searchConfig.RRFK)
		if err != nil {
			return nil, err
		}
		if len(results) > searchConfig.RerankCandidateK {
			results = results[:searchConfig.RerankCandidateK]
		}
		return results, nil
	}))
	_ = g.AddLambdaNode(Reranker, compose.InvokableLambda(func(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
		var query string
		if err := compose.ProcessState[*RetrievalState](ctx, func(ctx context.Context, state *RetrievalState) error {
			query = state.Query
			return nil
		}); err != nil {
			return nil, err
		}
		return resultReranker.Rerank(ctx, query, docs, searchConfig.FinalTopK)
	}))

	_ = g.AddEdge(compose.START, Trans_String)
	_ = g.AddEdge(Trans_String, ESRetriever)
	_ = g.AddEdge(Trans_String, MilvusRetriever)
	_ = g.AddEdge(ESRetriever, Fusion)
	_ = g.AddEdge(MilvusRetriever, Fusion)
	_ = g.AddEdge(Fusion, Reranker)
	_ = g.AddEdge(Reranker, compose.END)
	return g, nil
}
