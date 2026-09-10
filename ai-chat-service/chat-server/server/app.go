package server

import (
	chat_context "ai-chat-service/chat-server/chat-context"
	chat_round "ai-chat-service/chat-server/chat-round"
	"ai-chat-service/chat-server/chat-round/embedding"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	"ai-chat-service/pkg/zerror"
	"ai-chat-service/proto"
	"ai-chat-service/services"
	keywords_filter "ai-chat-service/services/keywords-filter"
	keywords_proto "ai-chat-service/services/keywords-filter/proto"
	"ai-chat-service/services/tokenizer"
	"context"
	"time"

	"github.com/google/uuid"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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
	ContextTTL        int
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
	log        log.ILogger
	// TODO 内容上下文对象
	contextCache chat_context.ContextCache
}

func (s *chatService) newApp(in *proto.ChatCompletionRequest, contextCache chat_context.ContextCache) *app {
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

		EmbeddingApiKey:           s.config.Embedding.ApiKey,
		EmbeddingBaseUrl:          s.config.Embedding.BaseUrl,
		EmbeddingVectorDimensions: s.config.Embedding.VectorDimensions,
		EmbeddingModel:            s.config.Embedding.Model,
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
		if in.ChatParam.ContextTTL != 0 {
			conf.ContextTTL = int(in.ChatParam.ContextTTL)
		}
		if in.ChatParam.ContextLen != 0 {
			conf.ContextLen = int(in.ChatParam.ContextLen)
		}
		if in.ChatParam.MinResponseTokens != 0 {
			conf.MinResponseTokens = int(in.ChatParam.MinResponseTokens)
		}
	}
	return &app{
		openaiConf:   conf,
		log:          s.log,
		contextCache: contextCache,
	}
}

func (a *app) getOpenaiClientV3() openai.Client {
	return openai.NewClient(option.WithAPIKey(a.openaiConf.ApiKey), option.WithBaseURL(a.openaiConf.BaseUrl))
}

func (a *app) getEmbeddingResponse(texts []string) (*embedding.EmbeddingResponse, error) {

	client := chat_round.GetEmbeddingObject()
	return client.Get(texts)
}

func (a *app) buildChatCompletionRequestV3(in *proto.ChatCompletionRequest, stream bool) (params openai.ChatCompletionNewParams, tokens, currTokens int, currMessage chat_context.ChatMessageContent, err error) {

	currMessage = chat_context.ChatMessageContent{
		Role:    ChatMessageRoleUser,
		Content: in.Message,
	}
	var contextList []*chat_context.ChatMessage
	if in.EnableContext {
		contextList = a.getContext(in.Pid)
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
func (a *app) buildChatCompletionResponse(msg string, AnswerSource string, totalTokens int) *proto.ChatCompletionResponse {
	res := &proto.ChatCompletionResponse{
		Id:      uuid.New().String(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   a.openaiConf.Model,
		Choices: []*proto.ChatCompletionChoice{
			{
				Message: &proto.ChatCompletionMessage{
					Role:    ChatMessageRoleAssistant,
					Content: msg,
				},
				FinishReason: "stop",
			},
		},
		Usage: &proto.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      int32(totalTokens),
		},
		AnswerSource: AnswerSource,
	}
	return res
}

func (a *app) buildChatCompletionStreamResponse(id, delta, finishReason string) *proto.ChatCompletionStreamResponse {
	res := &proto.ChatCompletionStreamResponse{
		Id:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   a.openaiConf.Model,
		Choices: []*proto.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: &proto.ChatCompletionStreamChoiceDelta{
					Content: delta,
					Role:    ChatMessageRoleAssistant,
				},
				FinishReason: finishReason,
			},
		},
	}
	return res
}

func (a *app) buildAnswerMetaResponse(id, source string, tokenCount int) *proto.ChatCompletionStreamResponse {
	return &proto.ChatCompletionStreamResponse{
		Id:           id,
		AnswerSource: source,
		TokenCount:   int32(tokenCount),
	}
}

func (a *app) buildChatCompletionStreamResponseList(id, msg string) []*proto.ChatCompletionStreamResponse {
	list := make([]*proto.ChatCompletionStreamResponse, 0)
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

func (a *app) getContext(id string) []*chat_context.ChatMessage {
	maxLen := a.openaiConf.ContextLen
	list := make([]*chat_context.ChatMessage, 0, maxLen)
	key := id
	for i := 0; i < maxLen; i++ {
		value, err := a.contextCache.GetContext(key)
		if err != nil {
			a.log.Error(err)
			return nil
		}
		if value == nil {
			break
		}
		list = append(list, value)
		key = value.PID
	}
	return list
}
func (a *app) saveContext(value *chat_context.ChatMessage) error {
	err := a.contextCache.SetContext(value.ID, value)
	if err != nil {
		a.log.Error(err)
		return err
	}
	return nil
}

func (a *app) sensitive(in *proto.ChatCompletionRequest) (ok bool, msg string, err error) {
	pool := keywords_filter.GetSensitiveClientPool()
	conn := pool.Get()
	defer pool.Put(conn)
	accessToken := config.GetConfig().DependOn.Sensitive.AccessToken
	client := keywords_proto.NewFilterClient(conn)
	ctx := services.AppendBearerTokenToContext(context.Background(), accessToken)
	req := &keywords_proto.FilterReq{
		Text: in.Message,
	}
	res, err := client.Validate(ctx, req)
	if err != nil {
		a.log.Error(err)
		return false, "", err
	}
	ok = res.Ok
	if !ok {
		msg = "触发到了知识盲区，请换个问题再问"
	}
	return
}

func (a *app) textRetrieval(text string, vector []float32) (*chat_round.SemanticCacheEntry, error) {
	cacheClient, err := chat_round.NewKvstoreCache()
	if err != nil {
		return nil, err
	}
	defer cacheClient.Close()

	vectorIndex, err := chat_round.GetVectorIndex()
	if err != nil {
		return nil, err
	}

	round := chat_round.GetRoundObject(cacheClient, vectorIndex)

	return round.Retrieval(text, vector)
}

func (a *app) textUpdate(query string, newEntry *chat_round.SemanticCacheEntry) error {
	cacheClient, err := chat_round.NewKvstoreCache()
	if err != nil {
		return err
	}
	defer cacheClient.Close()

	vectorIndex, err := chat_round.GetVectorIndex()
	if err != nil {
		return err
	}
	round := chat_round.GetRoundObject(cacheClient, vectorIndex)

	return round.Update(query, newEntry)
}

func (a *app) replyStream(message string, stream proto.Chat_ChatCompletionStreamServer) error {
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

func (a *app) replyStreamWithMeta(message string, source string, tokenCount int, stream proto.Chat_ChatCompletionStreamServer) error {

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
