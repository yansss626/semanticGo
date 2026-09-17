package chat

type ChatCompletionStreamResponse struct {
	Id           string                        `json:"id,omitempty"`
	Object       string                        `json:"object,omitempty"`
	Model        string                        `json:"model,omitempty"`
	Choices      []*ChatCompletionStreamChoice `json:"choices,omitempty"`
	TokenCount   int32                         `json:"token_count,omitempty"`
	AnswerSource string                        `json:"answer_source,omitempty"`
	Created      int64                         `json:"created,omitempty"`
}

type ChatCompletionStreamChoice struct {
	Index        int32                           `json:"index,omitempty"`
	Delta        ChatCompletionStreamChoiceDelta `json:"delta,omitempty"`
	FinishReason string                          `json:"finish_reason,omitempty"`
}

type ChatCompletionResponse struct {
	Id           string                  `json:"id,omitempty"`
	Object       string                  `json:"object,omitempty"`
	Model        string                  `json:"model,omitempty"`
	Choices      []*ChatCompletionChoice `json:"choices,omitempty"`
	Usage        *Usage                  `json:"usage,omitempty"`
	AnswerSource string                  `json:"answer_source,omitempty"`
	Created      int64                   `json:"created,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int32                 `json:"index,omitempty"`
	Message      ChatCompletionMessage `json:"message,omitempty"`
	FinishReason string                `json:"finish_reason,omitempty"`
}

type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens,omitempty"`
	CompletionTokens int32 `json:"completion_tokens,omitempty"`
	TotalTokens      int32 `json:"total_tokens,omitempty"`
}

type ChatCompletionStreamChoiceDelta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

type ChatCompletionRequest struct {
	Message       string     `json:"message,omitempty"`
	Id            string     `json:"id,omitempty"`
	Pid           string     `json:"p_id,omitempty"`
	EnableContext bool       `json:"enable_context,omitempty"`
	ChatParam     *ChatParam `json:"chat_param,omitempty"`
}

type ChatParam struct {
	Model             string  `json:"model,omitempty"`
	MaxTokens         int32   `json:"max_tokens,omitempty"`
	TopP              float32 `json:"top_p,omitempty"`
	PresencePenalty   float32 `json:"presence_penalty,omitempty"`
	FrequencyPenalty  float32 `json:"frequency_penalty,omitempty"`
	BotDesc           string  `json:"bot_desc,omitempty"`
	MinResponseTokens int32   `json:"min_response_tokens,omitempty"`
	ContextTTL        int32   `json:"context_ttl,omitempty"`
	ContextLen        int32   `json:"context_len,omitempty"`
	Temperature       float32 `json:"temperature,omitempty"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Name    string `json:"name,omitempty"`
}
