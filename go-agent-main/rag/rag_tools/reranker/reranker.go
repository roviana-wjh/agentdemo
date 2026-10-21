package reranker

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/schema"
)

type Reranker interface {
	Rerank(ctx context.Context, query string, docs []*schema.Document, topK int) ([]*schema.Document, error)
}

type LexicalReranker struct{}

func NewLexicalReranker() *LexicalReranker {
	return &LexicalReranker{}
}

func (r *LexicalReranker) Rerank(ctx context.Context, query string, docs []*schema.Document, topK int) ([]*schema.Document, error) {
	if len(docs) == 0 {
		return []*schema.Document{}, nil
	}
	terms := tokenize(query)
	maxPrior := docs[0].Score()
	for _, doc := range docs[1:] {
		if doc.Score() > maxPrior {
			maxPrior = doc.Score()
		}
	}
	results := append([]*schema.Document(nil), docs...)
	for _, doc := range results {
		title, _ := doc.MetaData["title"].(string)
		text := strings.ToLower(title + " " + title + " " + doc.Content)
		matched := 0
		for term := range terms {
			if strings.Contains(text, term) {
				matched++
			}
		}
		lexicalScore := 0.0
		if len(terms) > 0 {
			lexicalScore = float64(matched) / float64(len(terms))
		}
		priorScore := 0.0
		if maxPrior > 0 {
			priorScore = doc.Score() / maxPrior
		}
		score := 0.7*priorScore + 0.3*lexicalScore
		doc.WithScore(score)
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		doc.MetaData["rerank_mode"] = "lexical"
		doc.MetaData["rerank_score"] = score
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score() > results[j].Score()
	})
	return limit(results, topK), nil
}

func tokenize(text string) map[string]struct{} {
	terms := make(map[string]struct{})
	var word []rune
	flush := func() {
		if len(word) > 0 {
			terms[strings.ToLower(string(word))] = struct{}{}
			word = word[:0]
		}
	}
	for _, current := range []rune(strings.ToLower(text)) {
		if unicode.Is(unicode.Han, current) {
			flush()
			terms[string(current)] = struct{}{}
			continue
		}
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			word = append(word, current)
		} else {
			flush()
		}
	}
	flush()
	return terms
}

func limit(docs []*schema.Document, topK int) []*schema.Document {
	if topK > 0 && len(docs) > topK {
		return docs[:topK]
	}
	return docs
}
