package reranker

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LocalRerank struct {
	BaseUrl        string
	Client         *http.Client
	ScoreThreshold float64
}

type LocalRerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type LocalRerankResponse struct {
	Results []RerankResult `json:"results"`
}

func (l *LocalRerank) BuildingRequest(query string, documents []string) *LocalRerankRequest {

	return &LocalRerankRequest{
		Query: query,

		Documents: documents,
	}
}

func (l *LocalRerank) SendingRequest(reqBody *LocalRerankRequest) (*LocalRerankResponse, error) {

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, l.BaseUrl, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("local reank returned status %d: %s", resp.StatusCode, body)
	}

	result := &LocalRerankResponse{}
	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (l *LocalRerank) ReRank(query string, results []*index_algorithm.SearchResult) (*index_algorithm.SearchResult, error) {
	if query == "" || len(results) == 0 {
		return nil, nil
	}

	documents := make([]string, len(results))
	for i := 0; i < len(results); i++ {
		documents[i] = results[i].Query
	}
	reqBody := l.BuildingRequest(
		query,
		documents,
	)

	resp, err := l.SendingRequest(reqBody)

	if err != nil {

		return nil, err
	}

	if len(resp.Results) == 0 {

		return nil, fmt.Errorf(
			"local rerank returned empty results",
		)
	}

	best := resp.Results[0]

	if best.Index < 0 || best.Index >= len(results) {
		return nil, fmt.Errorf("invalid rerank index: %d", best.Index)
	}

	if best.RelevanceScore < l.ScoreThreshold {

		return nil, nil
	}

	return results[best.Index], nil
}
