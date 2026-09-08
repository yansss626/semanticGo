package reranker

import index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"

type ReRanker interface {
	ReRank(query string, results []*index_algorithm.SemanticCacheRecallResult) (*index_algorithm.SemanticCacheRecallResult, error)
}
