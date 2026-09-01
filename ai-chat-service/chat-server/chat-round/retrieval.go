package chat_round

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"ai-chat-service/chat-server/chat-round/reranker"
	"ai-chat-service/pkg/config"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type cacheLidis struct {
	cache RoundCache
	index index_algorithm.VectorIndex
	rank  reranker.ReRanker
	k     int
}

func GetRoundObject(cache RoundCache, vectorIndex index_algorithm.VectorIndex) Round {

	cnf := config.GetConfig()

	//remote_rerank
	rerank := &reranker.AlibabaRerank{
		BaseUrl:  cnf.Rerank.BaseUrl,
		ApiKey:   cnf.Rerank.ApiKey,
		Model:    cnf.Rerank.Model,
		Instruct: cnf.Rerank.Instruct,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		ScoreThreshold: cnf.Rerank.RerankScore,
	}

	return &cacheLidis{
		cache: cache,
		rank:  rerank,
		k:     cnf.Rerank.TopK,
		index: vectorIndex,
	}
}

func (c *cacheLidis) Retrieval(query string, vector []float32) (string, error) {
	topKResults, err := c.index.SearchK(vector, c.k)
	if err != nil {
		return "", err
	}

	bestResults, err := c.rank.ReRank(query, topKResults)
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
