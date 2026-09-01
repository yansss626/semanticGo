package chat_round

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"encoding/json"
)

func (c *cacheLidis) Update(vector []float32, query string, answer string) error {

	cand := &index_algorithm.Candidate{
		Query:  query,
		Vector: vector,
		Answer: answer,
	}

	value, err := json.Marshal(cand)
	if err != nil {
		return err
	}

	_, err = c.cache.Set(query, string(value))

	err = c.index.Add(query, vector)
	if err != nil {
		return err
	}

	return err
}
