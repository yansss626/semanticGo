package semantic

import (
	"context"

	"encoding/json"

	mrpc "github.com/yansss626/go-mrpc/runtime"
)

type SemanticService interface {
	VGet(ctx context.Context, req *VGetRequest) (*VGetResponse, error)

	VSet(ctx context.Context, req *VSetRequest) (*VSetResponse, error)
}

func RegisterSemanticService(registry *mrpc.Registry, service SemanticService) error {

	registry.Register("SemanticService", "VGet",
		mrpc.MethodHandler{
			UnaryHandler: func(ctx context.Context, data json.RawMessage) (any, error) {
				req := &VGetRequest{}
				err := json.Unmarshal(data, req)
				if err != nil {
					return nil, err
				}
				return service.VGet(ctx, req)
			},
		},
	)

	registry.Register("SemanticService", "VSet",
		mrpc.MethodHandler{
			UnaryHandler: func(ctx context.Context, data json.RawMessage) (any, error) {
				req := &VSetRequest{}
				err := json.Unmarshal(data, req)
				if err != nil {
					return nil, err
				}
				return service.VSet(ctx, req)
			},
		},
	)

	return nil
}
