package chat_round

import (
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/db/redis"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
)

type algorithmSimhash struct {
	encoder         *simhashEncoder
	cache           RoundCache
	cosineThreshold float64
	distance        int
}

var (
	once  sync.Once
	plane [][]float64
)

func GetRetrievalObject(roundcache RoundCache) TextSearch {

	cnf := config.GetConfig()

	once.Do(func() {
		plane = generatehyperPlanefor64bits(cnf.Embedding.VectorDimensions)
	})

	return &algorithmSimhash{
		cache: roundcache,
		encoder: &simhashEncoder{
			vectorDimensions: cnf.Embedding.VectorDimensions,
			hyperPlane:       plane,
		},
		cosineThreshold: cnf.Embedding.CosineSimilarityThreshold,
		distance:        cnf.Embedding.Distance,
	}
}

func (a *algorithmSimhash) BuildIndex(vector []float32) uint64 {
	// for 64bits simhash
	return a.encoder.vectorToSimhash64(vector)
}

// retrieval
func (a *algorithmSimhash) GetRounds(round *RoundMessage) ([]*ChatRound, error) {

	// verify hamming distance
	keys := convertUint16ToString(a.encoder.splitUint64(round.Simhash))

	values := make([]*ChatRound, len(keys))
	for i := 0; i < len(keys); i++ {
		str, err := a.cache.Get(redis.GetKey(keys[i]))
		if err != nil {
			return nil, err
		}
		if str != "" {
			value := &ChatRound{}
			err := json.Unmarshal([]byte(str), value)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ChatRound: %w", err)
			}
			values[i] = value
		}
	}
	return values, nil
}

func (a *algorithmSimhash) SimilarTextSearch(rounds []*ChatRound, round *RoundMessage) (string, error) {

	// match
	var messages []*RoundMessage

	for i := 0; i < len(rounds); i++ {
		if rounds[i] == nil {
			continue
		}

		for j := 0; j < len(rounds[i].Messages); j++ {
			if a.encoder.calculateHammingDistanceFor64bits(round.Simhash, rounds[i].Messages[j].Simhash) <= a.distance {
				messages = append(messages, rounds[i].Messages[j])
			}
		}
	}

	// verify cosine similarity smaller than threshold

	var index int = -1
	threshold := a.cosineThreshold
	for i := 0; i < len(messages); i++ {
		res := a.encoder.cosineSimilarity(round.EmbeddingVector, messages[i].EmbeddingVector)
		if res >= threshold {
			threshold = res
			index = i
		}
	}

	if index < 0 {
		return "", nil
	}

	answer, err := a.cache.Get(messages[index].AnswerID)
	if err != nil {
		return "", err
	}
	round.AnswerID = messages[index].AnswerID

	return answer, nil
}

func (a *algorithmSimhash) InsertRound(rounds []*ChatRound, round *RoundMessage) error {

	keys := convertUint16ToString(a.encoder.splitUint64(round.Simhash))

	for i := 0; i < len(rounds); i++ {

		if rounds[i] == nil {
			value, err := json.Marshal(&ChatRound{
				Messages: []*RoundMessage{round},
			})
			if err != nil {
				return err
			}
			_, err = a.cache.Set(redis.GetKey(keys[i]), string(value))
			if err != nil {
				return err
			}
		} else {
			rounds[i].Messages = append(rounds[i].Messages, round)
			value, err := json.Marshal(rounds[i])
			if err != nil {
				return err
			}
			_, err = a.cache.Mod(redis.GetKey(keys[i]), string(value))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func convertUint16ToString(u []uint16) []string {

	s := make([]string, len(u))

	for i := 0; i < len(u); i++ {
		s[i] = strconv.FormatUint(uint64(u[i]), 16)
	}
	return s
}
