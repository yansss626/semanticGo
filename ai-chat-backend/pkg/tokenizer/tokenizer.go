package tokenizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type tokenInfo struct {
	Code  int    `json:"code"`
	Count int    `json:"num_tokens"`
	Msg   string `json:"msg"`
}

type chatMessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func GetTokenCount(message chatMessageContent, model string) (int, error) {
	url := fmt.Sprintf("http://192.168.239.161:3002/tokenizer/%s", model)
	info := tokenInfo{}
	if err := postJSON(url, &message, &info); err != nil {
		return 0, err
	}
	if info.Code != 200 {
		return 0, fmt.Errorf("%v", info.Msg)
	}
	return info.Count, nil
}

func postJSON(url string, requestData *chatMessageContent, responseData *tokenInfo) error {
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(responseData)
}
