package server

import (
	"context"
	mrpc_filter "keywords-filter/mrpc_generated/filter"
	"keywords-filter/pkg/filter"
)

type filterService struct {
	filter filter.IFilter
}

func NewFilterService(filter filter.IFilter) *filterService {
	return &filterService{
		filter: filter,
	}
}

func (s *filterService) Validate(ctx context.Context, in *mrpc_filter.FilterRequest) (*mrpc_filter.ValidateResponse, error) {
	ok, word := s.filter.Validate(in.Text)
	return &mrpc_filter.ValidateResponse{
		Ok:      ok,
		Keyword: word,
	}, nil
}

func (s *filterService) FindAll(_ context.Context, in *mrpc_filter.FilterRequest) (*mrpc_filter.FindAllResponse, error) {
	words := s.filter.FindAll(in.Text)
	return &mrpc_filter.FindAllResponse{
		Keywords: words,
	}, nil
}
