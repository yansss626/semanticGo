package semantic

import (
	"context"

	mrpc "github.com/yansss626/mrpc/runtime"
)

type SemanticServiceClient struct {
	client *mrpc.Client
}

func NewSemanticServiceClient(client *mrpc.Client) *SemanticServiceClient {
	return &SemanticServiceClient{
		client: client,
	}
}

func (c *SemanticServiceClient) VGet(ctx context.Context, req *VGetRequest) (*VGetResponse, error) {

	var resp VGetResponse
	err := c.client.CallUnary(ctx, "SemanticService", "VGet", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *SemanticServiceClient) VSet(ctx context.Context, req *VSetRequest) (*VSetResponse, error) {

	var resp VSetResponse
	err := c.client.CallUnary(ctx, "SemanticService", "VSet", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
