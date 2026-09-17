package filter

type FilterRequest struct {
	Text string `json:"text"`
}

type ValidateResponse struct {
	Ok bool `json:"ok"`
	Keyword string `json:"keyword"`
}

type FindAllResponse struct {
	Keywords []string `json:"keywords"`
}

