package chat

import (
	"context"

	"encoding/json"

	mrpc "github.com/yansss626/mrpc/runtime"
)

type ChatService interface {
	ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)

	ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest, stream *mrpc.StreamServer) error
}

func RegisterChatService(registry *mrpc.Registry, service ChatService) error {

	registry.Register("ChatService", "ChatCompletion",
		mrpc.MethodHandler{
			UnaryHandler: func(ctx context.Context, data json.RawMessage) (any, error) {
				req := &ChatCompletionRequest{}
				err := json.Unmarshal(data, req)
				if err != nil {
					return nil, err
				}
				return service.ChatCompletion(ctx, req)
			},
		},
	)

	registry.Register("ChatService", "ChatCompletionStream",
		mrpc.MethodHandler{
			StreamHandler: func(ctx context.Context, data json.RawMessage, stream *mrpc.StreamServer) error {
				req := &ChatCompletionRequest{}
				err := json.Unmarshal(data, req)
				if err != nil {
					return err
				}
				return service.ChatCompletionStream(ctx, req, stream)
			},
		},
	)

	return nil
}
