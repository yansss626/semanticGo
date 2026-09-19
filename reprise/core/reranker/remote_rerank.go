package reranker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	vector_index "reprise/core/vector-index"
)

type AlibabaRerank struct {
	BaseUrl         string
	ApiKey          string
	Model           string
	Instruct        string
	Client          *http.Client
	RerankThreshold float64
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

func (a *AlibabaRerank) Rerank(query string, recallResults *vector_index.RecallResult) (*TopCandidate, error) {

	if recallResults == nil || len(recallResults.Querys) == 0 {
		return nil, nil
	}

	reqBody := a.BuildingRequest(query, recallResults.Querys)
	resp, err := a.SendingRequest(reqBody)
	if err != nil {
		return nil, err
	}

	bestIndex := resp.Results[0].Index
	if bestIndex < 0 || bestIndex >= len(resp.Results) {
		return nil, fmt.Errorf("invalid rerank index: %d", bestIndex)
	}

	if resp.Results[0].RelevanceScore < a.RerankThreshold {
		return nil, nil
	}

	return &TopCandidate{
		Query:       recallResults.Querys[bestIndex],
		RecallScore: recallResults.Scores[bestIndex],
		RerankScore: resp.Results[0].RelevanceScore,
	}, nil
}
