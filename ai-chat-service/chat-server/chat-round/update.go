package chat_round

import (
	"encoding/json"
)

func (c *cacheLidis) Update(vector []float32, query string, answer string) error {

	newEntry := &SemanticCacheEntry{
		Query:  query,
		Answer: answer,
		Vector: vector,
	}

	value, err := json.Marshal(newEntry)
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
