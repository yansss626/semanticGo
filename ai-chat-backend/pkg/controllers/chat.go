package controllers

import (
	"ai-chat-backend/mrpc_generated/chat"
	"ai-chat-backend/pkg/config"
	"ai-chat-backend/pkg/log"
	ai_chat_service "ai-chat-backend/services/ai-chat-service"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

const (
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)

type ChatService struct {
	config                *config.Config
	log                   log.ILogger
	chatServiceClientPool *ai_chat_service.ChatServiceClientPool
}

type ChatCompletionParams struct {
	Model                 string        `json:"model"`
	MaxTokens             int           `json:"max_tokens,omitempty"`
	Temperature           float32       `json:"temperature,omitempty"`
	PresencePenalty       float32       `json:"presence_penalty,omitempty"`
	FrequencyPenalty      float32       `json:"frequency_penalty,omitempty"`
	ChatSessionTTL        time.Duration `json:"chat_session_ttl"`
	ChatMinResponseTokens int           `json:"chat_min_response_tokens"`
}

type ChatMessageRequest struct {
	Prompt  string                    `json:"prompt"`
	Options ChatMessageRequestOptions `json:"options"`
}

type ChatMessageRequestOptions struct {
	Name            string `json:"name"`
	ParentMessageId string `json:"parentMessageId"`
}

type ChatMessage struct {
	ID              string                             `json:"id"`
	Text            string                             `json:"text"`
	Role            string                             `json:"role"`
	Name            string                             `json:"name"`
	Delta           string                             `json:"delta"`
	Detail          *chat.ChatCompletionStreamResponse `json:"detail"`
	TokenCount      int                                `json:"tokenCount"`
	ParentMessageId string                             `json:"parentMessageId"`
	AnswerSource    string                             `json:"answerSource,omitempty"`
}

func NewChatService(config *config.Config, log log.ILogger, chatServiceClientPool *ai_chat_service.ChatServiceClientPool) (*ChatService, error) {
	return &ChatService{
		config:                config,
		log:                   log,
		chatServiceClientPool: chatServiceClientPool,
	}, nil
}

func (c *ChatService) ChatProcess(ctx *gin.Context) {
	payload := ChatMessageRequest{}
	if err := ctx.BindJSON(&payload); err != nil {
		klog.Error(err)
		ctx.JSON(200, gin.H{
			"status":  "Fail",
			"message": fmt.Sprintf("%v", err),
			"data":    nil,
		})
		return
	}

	messageID := uuid.New().String()

	result := ChatMessage{
		ID:              uuid.New().String(),
		Role:            ChatMessageRoleAssistant,
		Text:            "",
		ParentMessageId: messageID,
	}

	aiChatServiceClient, put, err := c.chatServiceClientPool.Get()
	if err != nil {
		c.log.Error(err)
		ctx.JSON(200, gin.H{
			"status":  "Fail",
			"message": fmt.Sprintf("%v", err),
			"data":    nil,
		})
		return
	}
	defer put()
	in := &chat.ChatCompletionRequest{
		Id:            messageID,
		Message:       payload.Prompt,
		Pid:           payload.Options.ParentMessageId,
		EnableContext: false,
		ChatParam: &chat.ChatParam{
			Model:             c.config.Chat.Model,
			MaxTokens:         int32(c.config.Chat.MaxTokens),
			Temperature:       c.config.Chat.Temperature,
			TopP:              c.config.Chat.TopP,
			PresencePenalty:   c.config.Chat.PresencePenalty,
			FrequencyPenalty:  c.config.Chat.FrequencyPenalty,
			BotDesc:           c.config.Chat.BotDesc,
			ContextTTL:        int32(c.config.Chat.ContextTTL),
			ContextLen:        int32(c.config.Chat.ContextLen),
			MinResponseTokens: int32(c.config.Chat.MinResponseTokens),
		},
	}
	if in.Pid != "" {
		in.EnableContext = true
	}

	stream, err := aiChatServiceClient.ChatCompletionStream(ctx, in)
	if err != nil {
		c.log.Error(err)
		ctx.JSON(200, gin.H{
			"status":  "Fail",
			"message": fmt.Sprintf("%v", err),
			"data":    nil,
		})
		return
	}

	firstChunk := true
	ctx.Header("Content-type", "application/octet-stream")
	for {
		result.Delta = ""
		rsp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return
		}

		if err != nil {
			klog.Error(err)
			ctx.JSON(200, gin.H{
				"status":  "Fail",
				"message": fmt.Sprintf("OpenAI Event Error %v", err),
				"data":    nil,
			})
			return
		}

		if rsp.Id != "" {
			result.ID = rsp.Id
		}

		if rsp.AnswerSource != "" {
			result.AnswerSource = rsp.AnswerSource
			result.TokenCount = int(rsp.TokenCount)
		}

		if len(rsp.Choices) > 0 {
			content := rsp.Choices[0].Delta.Content
			result.Delta = content
			if len(content) > 0 {
				result.Text += content
			}
			result.Detail = rsp
		}

		bts, err := json.Marshal(result)
		if err != nil {
			klog.Error(err)
			ctx.JSON(200, gin.H{
				"status":  "Fail",
				"message": fmt.Sprintf("OpenAI Event Marshal Error %v", err),
				"data":    nil,
			})
			return
		}

		if !firstChunk {
			ctx.Writer.Write([]byte("\n"))
		} else {
			firstChunk = false
		}

		if _, err := ctx.Writer.Write(bts); err != nil {
			klog.Error(err)
			return
		}

		ctx.Writer.Flush()
	}
}
