package semantic

import (
	"context"
	mrpc "github.com/yansss/mrpc/runtime"
)

type SemanticClient struct {
	client *mrpc.Client
}

func NewSemanticClient(client *mrpc.Client) *SemanticClient {
	return &SemanticClient{
		client: client,
	}
}

func (c *SemanticClient) VGet(ctx context.Context, req *VGetRequest) (*VGetResponse, error) {

	var resp VGetResponse
	err := c.client.CallUnary(ctx, "Semantic", "VGet", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *SemanticClient) VSet(ctx context.Context, req *VSetRequest) (*VSetResponse, error) {

	var resp VSetResponse
	err := c.client.CallUnary(ctx, "Semantic", "VSet", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
