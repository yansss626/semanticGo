package server

import (
	chat_context "ai-chat-service/chat-server/chat-context"
	"ai-chat-service/mrpc_generated/chat"
	"ai-chat-service/mrpc_generated/filter"
	"ai-chat-service/mrpc_generated/semantic"
	"ai-chat-service/pkg/zerror"
	keywords_filter "ai-chat-service/services/keywords-filter"
	"ai-chat-service/services/reprise"
	"ai-chat-service/services/tokenizer"
	"context"
	"time"

	"github.com/google/uuid"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	mrpc "github.com/yansss626/go-mrpc/runtime"
)

const ChatPrimedTokens = 2

const (
	AnswerSourcePublicModel  = "public_model"
	AnswerSourceCache        = "cache"
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)

type openaiConf struct {
	// LLM model
	ApiKey            string
	BaseUrl           string
	Model             string
	MaxTokens         int
	Temperature       float32
	TopP              float32
	PresencePenalty   float32
	FrequencyPenalty  float32
	BotDesc           string
	ContextLen        int
	MinResponseTokens int
	ThinkingType      string
	ReasoningEffort   string
	// Embedding Model
	EmbeddingApiKey           string
	EmbeddingBaseUrl          string
	EmbeddingVectorDimensions int
	EmbeddingModel            string
}
type app struct {
	openaiConf *openaiConf
	// TODO 内容上下文对象
	contextCache      chat_context.ContextCache
	filterClientPool  *keywords_filter.FilterClientPool
	repriseClientPool *reprise.RepriseClientPool
}

func (s *chatService) newApp(in *chat.ChatCompletionRequest, contextCache chat_context.ContextCache,
	filterClientPool *keywords_filter.FilterClientPool, repriseClientPool *reprise.RepriseClientPool) *app {
	conf := &openaiConf{
		ApiKey:            s.config.Chat.ApiKey,
		BaseUrl:           s.config.Chat.BaseUrl,
		Model:             s.config.Chat.Model,
		MaxTokens:         s.config.Chat.MaxTokens,
		Temperature:       s.config.Chat.Temperature,
		TopP:              s.config.Chat.TopP,
		PresencePenalty:   s.config.Chat.PresencePenalty,
		FrequencyPenalty:  s.config.Chat.FrequencyPenalty,
		BotDesc:           s.config.Chat.BotDesc,
		ContextLen:        s.config.Chat.ContextLen,
		MinResponseTokens: s.config.Chat.MinResponseTokens,
		ThinkingType:      s.config.Chat.ThinkingType,
		ReasoningEffort:   s.config.Chat.ReasoningEffort,
	}
	if in.ChatParam != nil {
		if in.ChatParam.Model != "" {
			conf.Model = in.ChatParam.Model
		}
		if in.ChatParam.TopP != 0 {
			conf.TopP = in.ChatParam.TopP
		}
		if in.ChatParam.FrequencyPenalty != 0 {
			conf.FrequencyPenalty = in.ChatParam.FrequencyPenalty
		}
		if in.ChatParam.PresencePenalty != 0 {
			conf.PresencePenalty = in.ChatParam.PresencePenalty
		}
		if in.ChatParam.Temperature != 0 {
			conf.Temperature = in.ChatParam.Temperature
		}
		if in.ChatParam.BotDesc != "" {
			conf.BotDesc = in.ChatParam.BotDesc
		}
		if in.ChatParam.MaxTokens != 0 {
			conf.MaxTokens = int(in.ChatParam.MaxTokens)
		}
		if in.ChatParam.ContextLen != 0 {
			conf.ContextLen = int(in.ChatParam.ContextLen)
		}
		if in.ChatParam.MinResponseTokens != 0 {
			conf.MinResponseTokens = int(in.ChatParam.MinResponseTokens)
		}
	}
	return &app{
		openaiConf:        conf,
		contextCache:      contextCache,
		filterClientPool:  filterClientPool,
		repriseClientPool: repriseClientPool,
	}
}

// 获取大模型客户端
func (a *app) getOpenaiClientV3() openai.Client {
	return openai.NewClient(option.WithAPIKey(a.openaiConf.ApiKey), option.WithBaseURL(a.openaiConf.BaseUrl))
}

// 构造大模型请求
func (a *app) buildChatCompletionRequestV3(in *chat.ChatCompletionRequest) (params openai.ChatCompletionNewParams, tokens, currTokens int, currMessage chat_context.ChatMessageContent,
	contextList []*chat_context.ChatMessage, err error) {

	currMessage = chat_context.ChatMessageContent{
		Role:    ChatMessageRoleUser,
		Content: in.Message,
	}
	if in.EnableContext {
		contextList, err = a.getContext(in.Pid)
		if err != nil {
			return
		}
	}
	tokens, currTokens, messages, err := a.rebuildMessages(contextList, currMessage)
	if err != nil {
		return
	}

	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {

		switch msg.Role {

		case ChatMessageRoleSystem:
			openaiMessages = append(
				openaiMessages,
				openai.SystemMessage(msg.Content),
			)

		case ChatMessageRoleAssistant:
			openaiMessages = append(
				openaiMessages,
				openai.AssistantMessage(msg.Content),
			)

		default:
			openaiMessages = append(
				openaiMessages,
				openai.UserMessage(msg.Content),
			)
		}
	}

	params = openai.ChatCompletionNewParams{
		Model:            openai.ChatModel(a.openaiConf.Model),
		Messages:         openaiMessages,
		MaxTokens:        openai.Int(int64(a.openaiConf.MaxTokens)),
		Temperature:      openai.Float(float64(a.openaiConf.Temperature)),
		TopP:             openai.Float(float64(a.openaiConf.TopP)),
		PresencePenalty:  openai.Float(float64(a.openaiConf.PresencePenalty)),
		FrequencyPenalty: openai.Float(float64(a.openaiConf.FrequencyPenalty)),
		ReasoningEffort:  openai.ReasoningEffort(a.openaiConf.ReasoningEffort),
	}

	params.SetExtraFields(map[string]any{
		"thinking": map[string]any{
			"type": a.openaiConf.ThinkingType,
		},
	})

	return
}

// 根据上下文重构请求
func (a *app) rebuildMessages(contextList []*chat_context.ChatMessage, currMessage chat_context.ChatMessageContent) (tokens, currTokens int, messages []chat_context.ChatMessageContent, err error) {

	// 用户 prompt
	messages = []chat_context.ChatMessageContent{currMessage}
	currTokens, err = tokenizer.GetTokens(&currMessage, a.openaiConf.Model)
	if err != nil {
		return
	}
	tokens += currTokens

	// 历史上下文
	for _, item := range contextList {
		if item == nil {
			continue
		}
		if tokens+
			item.Tokens+ChatPrimedTokens > a.openaiConf.MaxTokens-a.openaiConf.MinResponseTokens {
			break
		}
		messages = append(messages, item.Message)
		tokens += item.Tokens + ChatPrimedTokens
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// 系统 prompt

	if a.openaiConf.BotDesc != "" {
		sysMessage := chat_context.ChatMessageContent{
			Role:    ChatMessageRoleSystem,
			Content: a.openaiConf.BotDesc,
		}
		messages = append([]chat_context.ChatMessageContent{sysMessage}, messages...)
		sysTokens, err := tokenizer.GetTokens(&sysMessage, a.openaiConf.Model)
		if err != nil {
			return 0, 0, nil, err
		}
		tokens += sysTokens
	}

	remainTokens := a.openaiConf.MaxTokens - tokens
	if remainTokens < a.openaiConf.MinResponseTokens {
		err = zerror.NewByMsg("请求消息超限")
		return

	}

	return
}

// 构造非流式响应
func (a *app) buildChatCompletionResponse(msg string, AnswerSource string, totalTokens int32) *chat.ChatCompletionResponse {
	res := &chat.ChatCompletionResponse{
		Id:      uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   a.openaiConf.Model,
		Choices: []*chat.ChatCompletionChoice{
			{
				Message: chat.ChatCompletionMessage{
					Role:    ChatMessageRoleAssistant,
					Content: msg,
				},
				FinishReason: "stop",
			},
		},
		Usage: &chat.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      totalTokens,
		},
		AnswerSource: AnswerSource,
	}
	return res
}

// 构造流式响应
func (a *app) buildChatCompletionStreamResponse(id, delta, finishReason string) *chat.ChatCompletionStreamResponse {
	res := &chat.ChatCompletionStreamResponse{
		Id:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   a.openaiConf.Model,
		Choices: []*chat.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: chat.ChatCompletionStreamChoiceDelta{
					Content: delta,
					Role:    ChatMessageRoleAssistant,
				},
				FinishReason: finishReason,
			},
		},
	}
	return res
}

func convertChatCompletion(src *openai.ChatCompletion) *chat.ChatCompletionResponse {
	return &chat.ChatCompletionResponse{
		Id:      src.ID,
		Object:  string(src.Object),
		Model:   src.Model,
		Choices: convertChoice(src.Choices),
		Usage:   convertUsage(src.Usage),
		Created: src.Created,
	}
}

func convertChoice(src []openai.ChatCompletionChoice) []*chat.ChatCompletionChoice {
	choices := make([]*chat.ChatCompletionChoice, 0, len(src))
	for _, item := range src {
		choice := &chat.ChatCompletionChoice{
			Index:        int32(item.Index),
			Message:      convertMessage(item.Message),
			FinishReason: item.FinishReason,
		}
		choices = append(choices, choice)
	}
	return choices
}

func convertMessage(src openai.ChatCompletionMessage) chat.ChatCompletionMessage {
	return chat.ChatCompletionMessage{
		Role:    string(src.Role),
		Content: src.Content,
	}
}

func convertUsage(src openai.CompletionUsage) *chat.Usage {
	return &chat.Usage{
		PromptTokens:     int32(src.PromptTokens),
		CompletionTokens: int32(src.CompletionTokens),
		TotalTokens:      int32(src.TotalTokens),
	}
}

func convertChatCompletionChunk(src openai.ChatCompletionChunk) *chat.ChatCompletionStreamResponse {
	return &chat.ChatCompletionStreamResponse{
		Id:      src.ID,
		Object:  string(src.Object),
		Model:   src.Model,
		Created: src.Created,
		Choices: convertChatCompletionStreamChoice(src.Choices),
	}
}

func convertChatCompletionStreamChoice(src []openai.ChatCompletionChunkChoice) []*chat.ChatCompletionStreamChoice {
	choices := make([]*chat.ChatCompletionStreamChoice, 0, len(src))
	for _, item := range src {
		choice := &chat.ChatCompletionStreamChoice{
			Index:        int32(item.Index),
			FinishReason: item.FinishReason,
			Delta:        convertChatCompletionStreamChoiceDelta(item.Delta),
		}
		choices = append(choices, choice)
	}
	return choices
}

func convertChatCompletionStreamChoiceDelta(src openai.ChatCompletionChunkChoiceDelta) chat.ChatCompletionStreamChoiceDelta {
	return chat.ChatCompletionStreamChoiceDelta{
		Content: src.Content,
		Role:    src.Role,
	}
}

// 构造响应元数据
func (a *app) buildAnswerMetaResponse(id, source string, tokenCount int32) *chat.ChatCompletionStreamResponse {
	return &chat.ChatCompletionStreamResponse{
		Id:           id,
		AnswerSource: source,
		TokenCount:   tokenCount,
	}
}

// 构造缓存命中流式响应切片
func (a *app) buildChatCompletionStreamResponseList(id, msg string) []*chat.ChatCompletionStreamResponse {
	list := make([]*chat.ChatCompletionStreamResponse, 0)
	runes := []rune(msg)
	chunkSize := 35

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := runes[i:end]
		list = append(list, a.buildChatCompletionStreamResponse(id, string(chunk), ""))
	}
	return list
}

// 获取上下文
func (a *app) getContext(id string) ([]*chat_context.ChatMessage, error) {
	maxLen := a.openaiConf.ContextLen
	list := make([]*chat_context.ChatMessage, 0, maxLen)
	key := id
	for i := 0; i < maxLen; i++ {
		value, err := a.contextCache.GetContext(key)
		if err != nil {
			return nil, err
		}
		if value == nil {
			break
		}
		list = append(list, value)
		key = value.PID
	}
	return list, nil
}

// 保存上下文
func (a *app) saveContext(value *chat_context.ChatMessage) error {
	err := a.contextCache.SetContext(value.ID, value)
	if err != nil {
		return err
	}
	return nil
}

// 敏感词过滤
func (a *app) sensitive(in *chat.ChatCompletionRequest) (ok bool, msg string, err error) {
	client, put, err := a.filterClientPool.Get()
	if err != nil {
		return false, "", err
	}
	defer put()
	req := &filter.FilterRequest{
		Text: in.Message,
	}
	resp, err := client.Validate(context.Background(), req)
	if err != nil {
		return false, "", err
	}
	ok = resp.Ok
	if !ok {
		msg = "触发到了知识盲区，请换个问题再问"
	}
	return
}

// 裁剪多余上下文
func (a *app) delExcessContext(contextList []*chat_context.ChatMessage) error {
	maxLen := a.openaiConf.ContextLen
	totalLen := len(contextList)
	if maxLen <= 0 || totalLen <= maxLen {
		return nil
	}

	for i := maxLen; i < totalLen; i++ {
		if contextList[i] != nil && contextList[i].ID != "" {
			err := a.contextCache.DelContext(contextList[i].ID)
			if err != nil {
				return err
			}
		}
	}
	last := contextList[totalLen-1]
	if last == nil {
		return nil
	}
	key := last.PID
	for key != "" {
		value, err := a.contextCache.GetContext(key)
		if err != nil {
			return err
		}
		err = a.contextCache.DelContext(key)
		if err != nil {
			return err
		}
		if value == nil {
			break
		}
		key = value.PID
	}

	return nil
}

// 流式回复
func (a *app) replyStream(message string, stream *mrpc.StreamServer) error {
	resId := uuid.New().String()
	startRes := a.buildChatCompletionStreamResponse(resId, "", "")
	endRes := a.buildChatCompletionStreamResponse(resId, "", "stop")
	err := stream.Send(startRes)
	if err != nil {
		return err
	}
	resList := a.buildChatCompletionStreamResponseList(resId, message)
	for _, res := range resList {
		err = stream.Send(res)
		if err != nil {
			return err
		}
	}
	err = stream.Send(endRes)
	if err != nil {
		return err
	}
	return nil
}

// 流式回复 + 元数据
func (a *app) replyStreamWithMeta(message string, source string, tokenCount int32, stream *mrpc.StreamServer) error {

	resId := uuid.New().String()

	startRes := a.buildChatCompletionStreamResponse(resId, "", "")
	err := stream.Send(startRes)
	if err != nil {
		return err
	}
	resList := a.buildChatCompletionStreamResponseList(
		resId,
		message,
	)
	for _, res := range resList {
		if err := stream.Send(res); err != nil {
			return err
		}
	}
	endRes := a.buildChatCompletionStreamResponse(
		resId,
		"",
		"stop",
	)
	err = stream.Send(endRes)
	if err != nil {
		return err
	}

	ansMeta := a.buildAnswerMetaResponse(resId, source, tokenCount)
	err = stream.Send(ansMeta)
	if err != nil {
		return err
	}

	return nil
}

// 搜索语义缓存内容
func (a *app) searchSementicCache(in *chat.ChatCompletionRequest) (*semantic.VGetResponse, error) {
	client, put, err := a.repriseClientPool.Get()
	if err != nil {
		return nil, err
	}
	defer put()

	req := &semantic.VGetRequest{
		Text: in.Message,
	}
	resp, err := client.VGet(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// 语义缓存更新
func (a *app) addSemanticCache(in *chat.ChatCompletionRequest, answer string, totalTokens int32) error {
	client, put, err := a.repriseClientPool.Get()
	if err != nil {
		return err
	}
	defer put()

	req := &semantic.VSetRequest{
		Text:        in.Message,
		Answer:      answer,
		TotalTokens: totalTokens,
	}
	_, err = client.VSet(context.Background(), req)
	return err
}
