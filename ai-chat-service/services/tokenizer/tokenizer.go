package tokenizer

import (
	"ai-chat-service/pkg/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

type tokensInfo struct {
	Code   int    `json:"code"`
	Tokens int    `json:"num_tokens"`
	Msg    string `json:"msg"`
}

var httpClient = &http.Client{}

func GetTokens(message *openai.ChatCompletionMessage, model string) (int, error) {
	cnf := config.GetConfig()
	url := fmt.Sprintf("%s/tokenizer/%s", cnf.DependOn.Tokenizer.Address, model)
	info := &tokensInfo{}
	if err := postJSON(url, message, info); err != nil {
		return 0, err
	}
	return info.Tokens, nil
}

func postJSON(url string, reqData interface{}, respData interface{}) error {

	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Tokenizer HTTP Error] Status Code: %d, Response Body:\n%s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	if err := json.Unmarshal(bodyBytes, respData); err != nil {
		log.Printf("[Tokenizer JSON Unmarshal Error] Raw Body is:\n%s", string(bodyBytes))
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	return nil
}
