package reranker

import vector_index "reprise/core/vector-index"

type TopCandidate struct {
	Query       string
	RecallScore float64
	RerankScore float64
}

type ReRanker interface {
	Rerank(query string, recallResults *vector_index.RecallResult) (*TopCandidate, error)
}
