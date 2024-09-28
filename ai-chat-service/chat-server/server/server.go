package server

import (
	chat_context "ai-chat-service/chat-server/chat-context"
	"ai-chat-service/chat-server/data"
	metrics_bus "ai-chat-service/chat-server/metrics-bus"
	vector_data "ai-chat-service/chat-server/vector-data"
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	"ai-chat-service/proto"
	"ai-chat-service/services/tokenizer"
	"context"
	"encoding/json"
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"io"
	"strconv"
	"strings"
	"time"
)

type chatService struct {
	proto.UnimplementedChatServer
	config     *config.Config
	log        log.ILogger
	data       data.IChatRecordsData
	vectorData vector_data.IChatRecordsData
	busMetrics *metrics_bus.BusMetrics
}

func NewChatService(data data.IChatRecordsData, vectorData vector_data.IChatRecordsData, config *config.Config, log log.ILogger, busMetrics *metrics_bus.BusMetrics) proto.ChatServer {
	return &chatService{
		config:     config,
		log:        log,
		data:       data,
		vectorData: vectorData,
		busMetrics: busMetrics,
	}
}

func (s *chatService) ChatCompletion(ctx context.Context, in *proto.ChatCompletionRequest) (*proto.ChatCompletionResponse, error) {
	redisContextCache := chat_context.NewRedisCache()
	defer redisContextCache.Close()

	app := s.newApp(in, redisContextCache)
	//敏感词过滤
	ok, msg, err := app.sensitive(in)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	if !ok {
		res := app.buildChatCompletionResponse(msg)
		return res, nil
	}

	//关键词提取
	keywords := app.keywords(in)
	if len(keywords) > 0 {
		idStr, score, err := s.vectorData.QueryData(context.Background(), map[string][]string{"keywords": {strings.Join(keywords, ",")}})
		if err != nil {
			s.log.Error(err)
		} else if score > 0.99 {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				s.log.Error(err)
			} else {
				record, err := s.data.GetById(id)
				if err != nil {
					s.log.Error(err)
				} else {
					res := app.buildChatCompletionResponse(record.AIMsg)
					return res, nil
				}
			}
		}
	}

	client := app.getOpenaiClient()
	req, tokens, currTokens, currMessage, err := app.buildChatCompletionRequest(in, false)
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	res := &proto.ChatCompletionResponse{}
	bytes, err := json.Marshal(resp)
	if err != nil {
		s.log.Error(err)
		return nil, err
	}
	err = jsonpb.UnmarshalString(string(bytes), res)
	if err != nil {
		s.log.Error(err)
		return nil, err
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
			ID:      resp.ID,
			PID:     reqContext.ID,
			Message: resp.Choices[0].Message,
			Tokens:  resp.Usage.CompletionTokens,
		}
		err = app.saveContext(resContext)
		if err != nil {
			s.log.Error(err)
			return
		}
	}()
	go func() {
		records := &data.ChatRecord{
			UserMsg:         in.Message,
			UserMsgTokens:   currTokens,
			UserMsgKeywords: keywords,
			AIMsg:           resp.Choices[0].Message.Content,
			AIMsgTokens:     resp.Usage.CompletionTokens,
			ReqTokens:       tokens,
			CreateAt:        time.Now().Unix(),
		}
		err := s.data.Add(records)
		if err != nil {
			s.log.Error(err)
			return
		}
		//保存到向量数据库
		if len(keywords) > 0 {
			list := []*vector_data.ChatRecord{
				{
					ID: strconv.FormatInt(records.ID, 10),
					KVs: map[string]string{
						"keywords": strings.Join(keywords, ","),
					},
				},
			}
			err = s.vectorData.UpsertData(context.Background(), list)
			if err != nil {
				s.log.Error(err)
				return
			}
		}
	}()
	return res, err
}
func (s *chatService) ChatCompletionStream(in *proto.ChatCompletionRequest, stream proto.Chat_ChatCompletionStreamServer) error {
	redisContextCache := chat_context.NewRedisCache()
	defer redisContextCache.Close()

	app := s.newApp(in, redisContextCache)
	//敏感词过滤
	ok, msg, err := app.sensitive(in)
	if err != nil {
		s.busMetrics.ErrQuestionsTotalCounter.Inc()
		s.log.Error(err)
		return err
	}
	if !ok {
		s.busMetrics.SensitiveQuestionsTotalCounter.Inc()
		resId := uuid.New().String()
		startRes := app.buildChatCompletionStreamResponse(resId, "", "")
		endRes := app.buildChatCompletionStreamResponse(resId, "", "stop")
		err = stream.Send(startRes)
		if err != nil {
			s.log.Error(err)
			return err
		}
		resList := app.buildChatCompletionStreamResponseList(resId, msg)
		for _, res := range resList {
			err = stream.Send(res)
			if err != nil {
				s.log.Error(err)
				return err
			}
		}
		err = stream.Send(endRes)
		if err != nil {
			s.log.Error(err)
			return err
		}
		return nil
	}

	//关键词提取
	keywords := app.keywords(in)

	if len(keywords) > 0 {
		s.busMetrics.KeywordsQuestionsTotalCounter.Inc()
		idStr, score, err := s.vectorData.QueryData(context.Background(), map[string][]string{"keywords": {strings.Join(keywords, ",")}})
		if err != nil {
			s.log.Error(err)
		} else if score > 0.99 {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				s.log.Error(err)
			} else {
				record, err := s.data.GetById(id)
				if err != nil {
					s.log.Error(err)
				} else {
					resId := uuid.New().String()
					startRes := app.buildChatCompletionStreamResponse(resId, "", "")
					endRes := app.buildChatCompletionStreamResponse(resId, "", "stop")
					err = stream.Send(startRes)
					if err != nil {
						s.log.Error(err)
						return err
					}
					resList := app.buildChatCompletionStreamResponseList(resId, record.AIMsg)
					for _, res := range resList {
						err = stream.Send(res)
						if err != nil {
							s.log.Error(err)
							return err
						}
					}
					err = stream.Send(endRes)
					if err != nil {
						s.log.Error(err)
						return err
					}
					return nil
				}
			}
		}
	}

	client := app.getOpenaiClient()
	req, tokens, currTokens, currMessage, err := app.buildChatCompletionRequest(in, false)
	chatStream, err := client.CreateChatCompletionStream(stream.Context(), req)
	if err != nil {
		s.busMetrics.ErrQuestionsTotalCounter.Inc()
		s.log.Error(err)
		return err
	}
	defer chatStream.Close()
	completionContent := ""
	resultID := ""
	for {
		resp, err := chatStream.Recv()
		if err != nil && err != io.EOF {
			s.log.Error(err)
			return err
		}
		if err == io.EOF {
			break
		}
		if resultID == "" {
			resultID = resp.ID
		}
		completionContent += resp.Choices[0].Delta.Content
		res := &proto.ChatCompletionStreamResponse{}
		bytes, err := json.Marshal(resp)
		if err != nil {
			s.log.Error(err)
			return err
		}
		err = jsonpb.UnmarshalString(string(bytes), res)
		if err != nil {
			s.log.Error(err)
			return err
		}
		err = stream.Send(res)
		if err != nil {
			s.log.Error(err)
			return err
		}
	}
	resultMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
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
	}()
	go func() {
		s.busMetrics.QuestionsTotalCounter.Inc()
		records := &data.ChatRecord{
			UserMsg:         in.Message,
			UserMsgTokens:   currTokens,
			UserMsgKeywords: keywords,
			AIMsg:           completionContent,
			AIMsgTokens:     resultTokens,
			ReqTokens:       tokens,
			CreateAt:        time.Now().Unix(),
		}
		err := s.data.Add(records)
		if err != nil {
			s.log.Error(err)
			return
		}
		//保存到向量数据库
		if len(keywords) > 0 {
			list := []*vector_data.ChatRecord{
				{
					ID: strconv.FormatInt(records.ID, 10),
					KVs: map[string]string{
						"keywords": strings.Join(keywords, ","),
					},
				},
			}
			err = s.vectorData.UpsertData(context.Background(), list)
			if err != nil {
				s.log.Error(err)
				return
			}
		}
	}()
	return nil
}
