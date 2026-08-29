package index_algorithm

import (
	"encoding/json"
	"fmt"
	"sort"
)

const defaultK = 3

type Cache interface {
	Get(key string) (string, error)
	Set(key, value string) (bool, error)
	Mod(key, value string) (bool, error)
}

type BruteForceSearch struct {
	Cache Cache
}

var _ VectorIndex = (*BruteForceSearch)(nil)

func (b *BruteForceSearch) Add(query string, vector []float32) error {

	key := "SemanticGo_candidates_index"

	str, err := b.Cache.Get(key)
	if err != nil {
		return err
	}

	if str == "" {

		querys := []string{query}
		value, err := json.Marshal(querys)
		if err != nil {
			return err
		}
		_, err = b.Cache.Set(key, string(value))
		if err != nil {
			return err
		}

	} else {

		var querys []string
		err = json.Unmarshal([]byte(str), &querys)
		if err != nil {
			return err
		}
		querys = append(querys, query)
		value, err := json.Marshal(querys)
		if err != nil {
			return err
		}
		_, err = b.Cache.Mod(key, string(value))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *BruteForceSearch) SearchK(vector []float32, k int) ([]*SearchResult, error) {

	queyrs, candidates, err := b.getCandidates()
	if err != nil {
		return nil, err
	}

	if k <= 0 {
		k = defaultK
	}

	if candidates == nil || queyrs == nil {
		return nil, nil
	}

	for i := 0; i < len(candidates); i++ {
		candidates[i].ReCallScore = CosineSimilarity(vector, candidates[i].Vector)
		candidates[i].Query = queyrs[i]
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].ReCallScore > candidates[j].ReCallScore
	})

	count := k
	if len(candidates) < k {
		count = len(candidates)
	}

	res := make([]*SearchResult, count)
	for i := 0; i < count; i++ {
		res[i] = &SearchResult{
			Query: candidates[i].Query,
			Score: candidates[i].ReCallScore,
		}
	}

	return res, nil
}

func (b *BruteForceSearch) Delete(query string) error {
	return nil
}

func (b *BruteForceSearch) getCandidates() ([]string, []*Candidate, error) {

	key := "SemanticGo_candidates_index"

	value, err := b.Cache.Get(key)
	if err != nil {
		return nil, nil, err
	}

	if value == "" {
		return nil, nil, nil
	}

	var querys []string
	err = json.Unmarshal([]byte(value), &querys)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	candidates := make([]*Candidate, 0, len(querys))
	validQueries := make([]string, 0, len(querys))
	for i := 0; i < len(querys); i++ {
		value, err = b.Cache.Get(querys[i])
		if err != nil {
			return nil, nil, err
		}

		if value == "" {
			continue
		}

		cand := &Candidate{}
		err = json.Unmarshal([]byte(value), cand)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal: %w", err)
		}
		candidates = append(candidates, cand)
		validQueries = append(validQueries, querys[i])
	}

	return validQueries, candidates, nil
}
