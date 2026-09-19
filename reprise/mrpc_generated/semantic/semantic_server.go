package semantic

import (
	"context"

	"encoding/json"

	mrpc "github.com/yansss/mrpc/runtime"
)

type Semantic interface {
	VGet(ctx context.Context, req *VGetRequest) (*VGetResponse, error)

	VSet(ctx context.Context, req *VSetRequest) (*VSetResponse, error)
}

func RegisterSemantic(registry *mrpc.Registry, service Semantic) error {

	registry.Register("Semantic", "VGet",
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

	registry.Register("Semantic", "VSet",
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
