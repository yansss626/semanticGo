package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Qwen3TextEmbedding struct {
	Model           string
	Dim             int
	Client          *http.Client
	ApiKey          string
	BaseUrl         string
	TopK            int
	RecallThreshold float64
}

type Qwen3TextEmbeddingRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions"`
}

func (q *Qwen3TextEmbedding) BuildingRequest(texts []string) *Qwen3TextEmbeddingRequest {
	return &Qwen3TextEmbeddingRequest{
		Model:      q.Model,
		Input:      texts,
		Dimensions: q.Dim,
	}
}

func (q *Qwen3TextEmbedding) SendingRequest(reqBody *Qwen3TextEmbeddingRequest) (*EmbeddingResponse, error) {

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", q.BaseUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer"+q.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding API returned status %d: %s\n",
			resp.StatusCode,
			string(body))
	}

	embeddingResp := &EmbeddingResponse{}
	err = json.Unmarshal(body, embeddingResp)
	if err != nil {
		return nil, err
	}

	return embeddingResp, nil
}

func (q *Qwen3TextEmbedding) Get(texts []string) (*EmbeddingResponse, error) {

	req := q.BuildingRequest(texts)
	return q.SendingRequest(req)
}
