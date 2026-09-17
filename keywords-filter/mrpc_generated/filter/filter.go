package filter

type FindAllResponse struct {
	Keywords []string `json:"keywords"`
}

type FilterRequest struct {
	Text string `json:"text"`
}

type ValidateResponse struct {
	Ok bool `json:"ok"`
	Keyword string `json:"keyword"`
}

