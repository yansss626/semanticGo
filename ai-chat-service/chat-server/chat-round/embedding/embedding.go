package embedding

type EmbeddingResponse struct {
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Object string      `json:"object"`
	Usage  Usage       `json:"usage"`
}

type Embedding struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
	Object    string    `json:"object"`
}

type Usage struct {
	TotalTokens int64 `json:"total_tokens"`
}

type EmbeddingObject interface {
	Get(texts []string) (*EmbeddingResponse, error)
}
