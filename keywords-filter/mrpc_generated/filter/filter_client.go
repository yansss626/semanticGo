package filter

import (
	"context"
	mrpc "github.com/yansss/mrpc/runtime"
)

type FilterServiceClient struct {
	client *mrpc.Client
}

func NewFilterServiceClient(client *mrpc.Client) *FilterServiceClient {
	return &FilterServiceClient{
		client: client,
	}
}

func (c *FilterServiceClient) FindAll(ctx context.Context, req *FilterRequest) (*FindAllResponse, error) {

	var resp FindAllResponse
	err := c.client.CallUnary(ctx, "FilterService", "FindAll", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FilterServiceClient) Validate(ctx context.Context, req *FilterRequest) (*ValidateResponse, error) {

	var resp ValidateResponse
	err := c.client.CallUnary(ctx, "FilterService", "Validate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
