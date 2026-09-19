package semantic

type VSetResponse struct {
}

type VGetRequest struct {
	Text string `json:"text,omitempty"`
}

type VGetResponse struct {
	Answer string `json:"answer,omitempty"`
	TotalTokens int32 `json:"total_tokens,omitempty"`
}

type VSetRequest struct {
	Text string `json:"text,omitempty"`
	Answer string `json:"answer,omitempty"`
	TotalTokens int32 `json:"total_tokens,omitempty"`
}

