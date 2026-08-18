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
	if info.Code != 200 {
		return 0, fmt.Errorf("%v", info.Msg)
	}
	return info.Tokens, nil
}

//	func postJSON(url string, requestData *openai.ChatCompletionMessage, responseData *tokensInfo) error {
//		requestBody, err := json.Marshal(requestData)
//		if err != nil {
//			return err
//		}
//		resp, err := httpClient.Post(url, "application/json", bytes.NewReader(requestBody))
//		if err != nil {
//			return err
//		}
//		return json.NewDecoder(resp.Body).Decode(responseData)
//	}
func postJSON(url string, reqData interface{}, respData interface{}) error {
	// 1. 序列化请求体
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	// 2. 发起 POST 请求
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("http post failed: %w", err)
	}
	defer resp.Body.Close()

	// 3. 读取原始 Body 字节流
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}

	// 4. 检查 HTTP 状态码，如果不为 200，直接打印原始 HTML 内容
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Tokenizer HTTP Error] Status Code: %d, Response Body:\n%s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	// 5. 尝试解析 JSON，如果解析报错（如遇到 '<'），同样打印出原始 Body
	if err := json.Unmarshal(bodyBytes, respData); err != nil {
		log.Printf("[Tokenizer JSON Unmarshal Error] Raw Body is:\n%s", string(bodyBytes))
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	return nil
}
