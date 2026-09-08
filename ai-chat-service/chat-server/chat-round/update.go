package chat_round

import (
	"encoding/json"
	"fmt"
)

func (c *cacheLidis) Update(query string, newCacheEntry *SemanticCacheEntry) error {

	if newCacheEntry == nil || newCacheEntry.Answer == "" ||
		newCacheEntry.Vector == nil || newCacheEntry.TotalTokens <= 0 {
		return fmt.Errorf("invalid cache entry for query [%s]: entry or required fields (answer, vector, totalTokens) are invalid", query)
	}

	value, err := json.Marshal(newCacheEntry)
	if err != nil {
		return err
	}

	_, err = c.cache.Set(query, string(value))

	err = c.index.Add(query, newCacheEntry.Vector)
	if err != nil {
		return err
	}

	return err
}
