package chat_round

import (
	"math"
	"math/bits"
	"math/rand"
)

type simhashEncoder struct {
	vectorDimensions int
	hyperPlane       [][]float64
	masks16Bits      []uint16
}

func generatehyperPlanefor64bits(vectorDimensions int) [][]float64 {

	const seed int64 = 4096

	r := rand.New(rand.NewSource(seed))

	HyperPlane := make([][]float64, 64)

	for i := 0; i < 64; i++ {
		row := make([]float64, vectorDimensions)
		for j := 0; j < vectorDimensions; j++ {
			row[j] = r.NormFloat64()
		}
		HyperPlane[i] = row
	}
	return HyperPlane
}

func generatemasksFor16bits(flipcount int) []uint16 {
	if flipcount <= 0 || flipcount > 4 {
		return nil
	}

	var masks []uint16
	for i := 0; i < 1<<16; i++ {
		if bits.OnesCount16(uint16(i)) == flipcount {
			masks = append(masks, uint16(i))
		}
	}

	return masks
}

func (s *simhashEncoder) cosineSimilarity(v1, v2 []float32) float64 {
	if len(v1) != s.vectorDimensions || len(v2) != s.vectorDimensions {
		return 0.0
	}

	var dotProduct, normV1, normV2 float64

	for i := 0; i < s.vectorDimensions; i++ {
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

func (s *simhashEncoder) vectorToSimhash64(vector []float32) uint64 {

	// random projection
	var simhash uint64

	for i := 0; i < 64; i++ {
		var dotProduct float64
		for j := 0; j < s.vectorDimensions; j++ {

			dotProduct += float64(vector[j]) * s.hyperPlane[i][j]
		}
		if dotProduct > 0 {
			simhash |= uint64(1) << i
		}

	}

	return simhash
}

func (s *simhashEncoder) calculateHammingDistanceFor64bits(simhash1, simhash2 uint64) int {

	return bits.OnesCount64(simhash1 ^ simhash2)
}

func (s *simhashEncoder) splitUint64ForUint16(x uint64) []uint16 {
	return []uint16{
		uint16(x & 0xFFFF),
		uint16(x >> 16 & 0xFFFF),
		uint16(x >> 32 & 0xFFFF),
		uint16(x >> 48 & 0xFFFF),
	}
}

func (s *simhashEncoder) uint16BitFlip(set []uint16) []uint16 {
	if s.masks16Bits == nil {
		return nil
	}

	result := make([]uint16, 0, len(set)*len(s.masks16Bits))

	for i := 0; i < len(set); i++ {
		for j := 0; j < len(s.masks16Bits); j++ {
			result = append(result, set[i]^s.masks16Bits[j])
		}
	}

	return result
}
