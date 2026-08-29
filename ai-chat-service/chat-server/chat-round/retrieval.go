package chat_round

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"encoding/json"
	"fmt"
)

type cacheLidis struct {
	cache RoundCache
	index index_algorithm.VectorIndex
	rank  reRanker
	k     int
}

func GetRoundObject(cache RoundCache) Round {
	return &cacheLidis{
		cache: cache,
	} // 待完善
}

type reRanker interface {
	reRank(query string, results []*index_algorithm.SearchResult) (*index_algorithm.SearchResult, error)
}

func (c *cacheLidis) Retrieval(query string, vector []float32) (string, error) {
	topKResults, err := c.index.SearchK(vector, c.k)
	if err != nil {
		return "", err
	}

	bestResults, err := c.rank.reRank(query, topKResults)
	if err != nil {
		return "", err
	}

	if bestResults == nil {
		return "", nil
	}

	value, err := c.cache.Get(bestResults.Query)
	if err != nil {
		return "", err
	}

	if value == "" {
		return "", nil
	}

	cand := &index_algorithm.Candidate{}
	err = json.Unmarshal([]byte(value), cand)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal Candidate: %w", err)
	}

	return cand.Answer, nil
}
