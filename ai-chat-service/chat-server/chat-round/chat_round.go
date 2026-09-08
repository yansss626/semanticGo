package chat_round

type SemanticCacheEntry struct {
	Answer string `json:"answer"`

	TotalTokens int `json:"total_tokens"`

	Vector []float32 `json:"vector"`
}

type Round interface {
	Retrieval(query string, vector []float32) (*SemanticCacheEntry, error)
	Update(query string, newCacheEntry *SemanticCacheEntry) error
}
