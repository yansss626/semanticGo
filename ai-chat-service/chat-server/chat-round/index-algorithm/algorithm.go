package index_algorithm

import "math"

type SemanticCacheRecallResult struct {
	Query string
	Score float64
}

type VectorIndex interface {
	Add(query string, vector []float32) error
	SearchK(vector []float32, k int) ([]*SemanticCacheRecallResult, error)
	Delete(query string) error
}

func CosineSimilarity(v1, v2 []float32) float64 {

	if len(v1) != len(v2) {
		return 0.0
	}

	var dotProduct, normV1, normV2 float64

	for i := 0; i < len(v1); i++ {
		val1 := float64(v1[i])
		val2 := float64(v2[i])

		dotProduct += val1 * val2
		normV1 += val1 * val1
		normV2 += val2 * val2
	}

	if normV1 == 0 || normV2 == 0 {
		return 0.0
	}

	return (dotProduct / (math.Sqrt(normV1) * math.Sqrt(normV2)))
}
