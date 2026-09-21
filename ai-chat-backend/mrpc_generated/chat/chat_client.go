package chat

import (
	"context"

	mrpc "github.com/yansss626/go-mrpc/runtime"
)

type ChatServiceClient struct {
	client *mrpc.Client
}

func NewChatServiceClient(client *mrpc.Client) *ChatServiceClient {
	return &ChatServiceClient{
		client: client,
	}
}

func (c *ChatServiceClient) ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {

	var resp ChatCompletionResponse
	err := c.client.CallUnary(ctx, "ChatService", "ChatCompletion", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type ChatServiceChatCompletionStreamClient struct {
	stream *mrpc.ClientStream
}

func (c *ChatServiceClient) ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (*ChatServiceChatCompletionStreamClient, error) {
	stream, err := c.client.NewClientStream(ctx, "ChatService", "ChatCompletionStream", req)
	if err != nil {
		return nil, err
	}
	return &ChatServiceChatCompletionStreamClient{
		stream: stream,
	}, nil
}

func (s *ChatServiceChatCompletionStreamClient) Recv() (*ChatCompletionStreamResponse, error) {
	var resp ChatCompletionStreamResponse
	err := s.stream.Recv(&resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
