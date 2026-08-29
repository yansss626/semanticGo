package reranker

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// rerank
type AlibabaRerank struct {
	BaseUrl        string
	ApiKey         string
	Model          string
	Instruct       string
	Client         *http.Client
	ScoreThreshold float64
}

type Qwen3RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	Instruct  string   `json:"instruct"`
}

type Qwen3RerankResponse struct {
	Object  string           `json:"object"`
	Results []RerankResult   `json:"results"`
	Model   string           `json:"model"`
	Id      string           `json:"id"`
	Usage   Qwen3RerankUsage `json:"usage"`
}

type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type Qwen3RerankUsage struct {
	TotalTokens int `json:"total_tokens"`
}

func (a *AlibabaRerank) BuildingRequest(query string, documents []string) *Qwen3RerankRequest {
	return &Qwen3RerankRequest{
		Model:     a.Model,
		Query:     query,
		Documents: documents,
		Instruct:  a.Instruct,
	}
}

func (a *AlibabaRerank) SendingRequest(reqBody *Qwen3RerankRequest) (*Qwen3RerankResponse, error) {

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}

	req, err := http.NewRequest("POST", a.BaseUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falied to sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rerank API returned status %d: %s",
			resp.StatusCode,
			string(body))
	}

	rerankResp := &Qwen3RerankResponse{}
	err = json.Unmarshal(body, rerankResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return rerankResp, nil
}

func (a *AlibabaRerank) ReRank(query string, results []*index_algorithm.SearchResult) (*index_algorithm.SearchResult, error) {

	if len(results) == 0 {
		return nil, nil
	}

	documents := make([]string, len(results))
	for i := 0; i < len(results); i++ {
		documents[i] = results[i].Query
	}

	reqBody := a.BuildingRequest(query, documents)
	resp, err := a.SendingRequest(reqBody)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("rerank returned no results")
	}

	index := resp.Results[0].Index
	if index < 0 || index >= len(resp.Results) {
		return nil, fmt.Errorf("invalid rerank index: %d", index)
	}

	if resp.Results[0].RelevanceScore < a.ScoreThreshold {
		return nil, nil
	}

	return results[index], nil
}
