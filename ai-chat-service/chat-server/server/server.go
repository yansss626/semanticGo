package server

import (
	chat_context "ai-chat-service/chat-server/chat-context"
	metrics_bus "ai-chat-service/chat-server/metrics-bus"
	"ai-chat-service/mrpc_generated/chat"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	keywords_filter "ai-chat-service/services/keywords-filter"
	"ai-chat-service/services/tokenizer"
	"context"

	mrpc "github.com/yansss/mrpc/runtime"
)

type chatService struct {
	config           *config.Config
	log              log.ILogger
	busMetrics       *metrics_bus.BusMetrics
	contextCache     chat_context.ContextCache
	filterClientPool *keywords_filter.FilterClientPool
}

func NewChatService(config *config.Config, log log.ILogger, busMetrics *metrics_bus.BusMetrics) (*chatService, error) {

	// 初始化敏感词/关键词服务 连接池
	filterClientPool, err := keywords_filter.InitFilterClientPool(config)
	if err != nil {
		return nil, err
	}

	// 初始化上下文缓存
	contextCache, err := chat_context.NewContextCache(config)
	if err != nil {
		return nil, err
	}

	return &chatService{
		config:           config,
		log:              log,
		busMetrics:       busMetrics,
		contextCache:     contextCache,
		filterClientPool: filterClientPool,
	}, nil
}

func (s *chatService) ChatCompletion(ctx context.Context, in *chat.ChatCompletionRequest) (*chat.ChatCompletionResponse, error) {

	app := s.newApp(in, s.contextCache, s.filterClientPool)
	//敏感词过滤
	ok, msg, err := app.sensitive(in)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	if !ok {
		res := app.buildChatCompletionResponse(msg, "", 0)
		return res, nil
	}

	client := app.getOpenaiClientV3()
	params, _, currTokens, currMessage, contextList, err := app.buildChatCompletionRequestV3(in)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	res := convertChatCompletion(resp)
	res.AnswerSource = AnswerSourcePublicModel

	// 保存上下文
	go func() {
		reqContext := &chat_context.ChatMessage{
			ID:      in.Id,
			PID:     in.Pid,
			Message: currMessage,
			Tokens:  currTokens,
		}
		err := app.saveContext(reqContext)
		if err != nil {
			s.log.Error(err)
			return
		}
		resContext := &chat_context.ChatMessage{
			ID:  resp.ID,
			PID: reqContext.ID,
			Message: chat_context.ChatMessageContent{
				Role:    ChatMessageRoleAssistant,
				Content: resp.Choices[0].Message.Content,
			},
			Tokens: int(resp.Usage.CompletionTokens),
		}
		err = app.saveContext(resContext)
		if err != nil {
			s.log.Error(err)
			return
		}
		newContextlist := []*chat_context.ChatMessage{reqContext, resContext}
		newContextlist = append(newContextlist, contextList...)
		err = app.delExcessContext(newContextlist)
		if err != nil {
			s.log.Error(err)
			return
		}
	}()

	return res, err
}
func (s *chatService) ChatCompletionStream(ctx context.Context, in *chat.ChatCompletionRequest, stream *mrpc.StreamServer) error {

	app := s.newApp(in, s.contextCache, s.filterClientPool)
	//敏感词过滤
	ok, msg, err := app.sensitive(in)
	if err != nil {
		s.busMetrics.ErrQuestionsTotalCounter.Inc()
		s.log.Error(err)
		return err
	}
	if !ok {
		s.busMetrics.SensitiveQuestionsTotalCounter.Inc()
		err = app.replyStreamWithMeta(msg, "", 0, stream)
		if err != nil {
			s.log.Error(err)
			return err
		}
		return nil
	}

	client := app.getOpenaiClientV3()
	params, tokens, currTokens, currMessage, contextList, err := app.buildChatCompletionRequestV3(in)
	if err != nil {
		s.log.Error(err)
		return err
	}
	chatStream := client.Chat.Completions.NewStreaming(ctx, params)
	defer chatStream.Close()

	completionContent := ""
	resultID := ""

	for chatStream.Next() {
		resp := chatStream.Current()
		if len(resp.Choices) == 0 {
			continue
		}
		if resultID == "" {
			resultID = resp.ID
		}
		content := resp.Choices[0].Delta.Content
		completionContent += content

		res := convertChatCompletionChunk(resp)
		err = stream.Send(res)
		if err != nil {
			s.log.Error(err)
			return err
		}
	}
	if err := chatStream.Err(); err != nil {
		s.busMetrics.ErrQuestionsTotalCounter.Inc()
		s.log.Error(err)
		return err
	}

	resultMessage := chat_context.ChatMessageContent{
		Role:    ChatMessageRoleAssistant,
		Content: completionContent,
	}
	model := s.config.Chat.Model
	if in.ChatParam != nil && in.ChatParam.Model != "" {
		model = in.ChatParam.Model
	}
	resultTokens, err := tokenizer.GetTokens(&resultMessage, model)
	if err != nil {
		s.busMetrics.ErrQuestionsTotalCounter.Inc()
		s.log.Error(err)
		return err
	}

	totalTokens := resultTokens + tokens
	ansMeta := app.buildAnswerMetaResponse(resultID, AnswerSourcePublicModel, totalTokens)
	err = stream.Send(ansMeta)
	if err != nil {
		s.log.Error(err)
		return err
	}

	go func() {
		reqContext := &chat_context.ChatMessage{
			ID:      in.Id,
			PID:     in.Pid,
			Message: currMessage,
			Tokens:  currTokens,
		}
		err := app.saveContext(reqContext)
		if err != nil {
			s.log.Error(err)
			return
		}
		resContext := &chat_context.ChatMessage{
			ID:      resultID,
			PID:     reqContext.ID,
			Message: resultMessage,
			Tokens:  resultTokens,
		}
		err = app.saveContext(resContext)
		if err != nil {
			s.log.Error(err)
			return
		}
		newContextlist := []*chat_context.ChatMessage{resContext, reqContext}
		newContextlist = append(newContextlist, contextList...)
		err = app.delExcessContext(newContextlist)
		if err != nil {
			s.log.Error(err)
			return
		}
	}()

	return nil
}
