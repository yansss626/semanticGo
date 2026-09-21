package server

import (
	chat_context "ai-chat-service/chat-server/chat-context"
	metrics_bus "ai-chat-service/chat-server/metrics-bus"
	"ai-chat-service/mrpc_generated/chat"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	keywords_filter "ai-chat-service/services/keywords-filter"
	"ai-chat-service/services/reprise"
	"ai-chat-service/services/tokenizer"
	"context"

	mrpc "github.com/yansss626/mrpc/runtime"
)

type chatService struct {
	config            *config.Config
	log               log.ILogger
	busMetrics        *metrics_bus.BusMetrics
	contextCache      chat_context.ContextCache
	filterClientPool  *keywords_filter.FilterClientPool
	repriseClientPool *reprise.RepriseClientPool
}

func NewChatService(config *config.Config, log log.ILogger, busMetrics *metrics_bus.BusMetrics) (*chatService, error) {

	// 初始化敏感词/关键词服务 连接池
	filterClientPool, err := keywords_filter.InitFilterClientPool(config)
	if err != nil {
		return nil, err
	}

	// 初始化语义缓存服务 连接池
	repriseClientPool, err := reprise.InitRepriseClientPool(config)
	if err != nil {
		return nil, err
	}

	// 初始化上下文缓存
	contextCache, err := chat_context.NewContextCache(config)
	if err != nil {
		return nil, err
	}

	return &chatService{
		config:            config,
		log:               log,
		busMetrics:        busMetrics,
		contextCache:      contextCache,
		filterClientPool:  filterClientPool,
		repriseClientPool: repriseClientPool,
	}, nil
}

func (s *chatService) ChatCompletion(ctx context.Context, in *chat.ChatCompletionRequest) (*chat.ChatCompletionResponse, error) {

	app := s.newApp(in, s.contextCache, s.filterClientPool, s.repriseClientPool)
	// 敏感词过滤
	ok, msg, err := app.sensitive(in)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	if !ok {
		res := app.buildChatCompletionResponse(msg, "", 0)
		return res, nil
	}

	// 查询语义缓存，是否能够复用缓存答案？
	semanticResp, err := app.searchSementicCache(in)
	if err != nil {
		s.log.Error(err)
	}
	if semanticResp != nil && semanticResp.Answer != "" { // 如果能复用，直接返回，不走大模型
		res := app.buildChatCompletionResponse(semanticResp.Answer, AnswerSourceCache, semanticResp.TotalTokens)
		return res, nil
	}

	// 访问大模型：大模型服务
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
	res.AnswerSource = AnswerSourcePublicModel // 标注大模型回答

	// 语义缓存更新
	go func() {
		err := app.addSemanticCache(in, res.Choices[0].Message.Content, res.Usage.TotalTokens)
		if err != nil {
			s.log.Error()
			return
		}
	}()

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

	app := s.newApp(in, s.contextCache, s.filterClientPool, s.repriseClientPool)
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

	// 查询语义缓存，是否能够复用缓存答案？
	semanticResp, err := app.searchSementicCache(in)
	if err != nil {
		s.log.Error(err)
	}
	if semanticResp != nil && semanticResp.Answer != "" {
		err := app.replyStreamWithMeta(semanticResp.Answer, AnswerSourceCache, semanticResp.TotalTokens, stream)
		if err != nil {
			s.log.Error(err)
			return err
		}
		return nil // 如果能复用，直接返回，不走大模型
	}

	// 访问大模型：大模型服务 (流式响应)
	// params：访问大模型的请求； tokens：请求总 token（包括上下文 + prompt）；currTokens：prompt token，
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

	// 计算响应 token
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

	totalTokens := resultTokens + tokens                                                          // 计算耗费总 token
	ansMeta := app.buildAnswerMetaResponse(resultID, AnswerSourcePublicModel, int32(totalTokens)) // 标注大模型回答
	err = stream.Send(ansMeta)
	if err != nil {
		s.log.Error(err)
		return err
	}

	// 语义缓存更新
	go func() {
		err := app.addSemanticCache(in, completionContent, int32(totalTokens))
		if err != nil {
			s.log.Error()
			return
		}
	}()

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
