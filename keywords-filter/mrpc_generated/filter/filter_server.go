package filter

import (
	"context"

	"encoding/json"

	mrpc "github.com/yansss626/go-mrpc/runtime"
)

type FilterService interface {
	FindAll(ctx context.Context, req *FilterRequest) (*FindAllResponse, error)

	Validate(ctx context.Context, req *FilterRequest) (*ValidateResponse, error)
}

func RegisterFilterService(registry *mrpc.Registry, service FilterService) error {

	registry.Register("FilterService", "FindAll",
		mrpc.MethodHandler{
			UnaryHandler: func(ctx context.Context, data json.RawMessage) (any, error) {
				req := &FilterRequest{}
				err := json.Unmarshal(data, req)
				if err != nil {
					return nil, err
				}
				return service.FindAll(ctx, req)
			},
		},
	)

	registry.Register("FilterService", "Validate",
		mrpc.MethodHandler{
			UnaryHandler: func(ctx context.Context, data json.RawMessage) (any, error) {
				req := &FilterRequest{}
				err := json.Unmarshal(data, req)
				if err != nil {
					return nil, err
				}
				return service.Validate(ctx, req)
			},
		},
	)

	return nil
}
