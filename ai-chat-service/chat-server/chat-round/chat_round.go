package chat_round

type SemanticCacheEntry struct {
	Query string `json:"-"`

	Answer string `json:"answer"`

	PromptTokens int `json:"prompt_tokens"`

	CompletionTokens int `json:"completion_tokens"`

	TotalTokens int `json:"total_tokens"`

	Vector []float32 `json:"vector"`
}

type Round interface {
	Retrieval(query string, vector []float32) (string, error)
	Update(vector []float32, query string, answer string) error
}
