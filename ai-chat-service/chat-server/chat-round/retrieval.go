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
	masks []uint16
)

func GetRetrievalObject(roundcache RoundCache) TextSearch {

	cnf := config.GetConfig()

	once.Do(func() {
		plane = generatehyperPlanefor64bits(cnf.Embedding.VectorDimensions)
		masks = generatemasksFor16bits(2)
	})

	return &algorithmSimhash{
		cache: roundcache,
		encoder: &simhashEncoder{
			vectorDimensions: cnf.Embedding.VectorDimensions,
			hyperPlane:       plane,
			masks16Bits:      masks,
		},
		cosineThreshold: cnf.Embedding.CosineSimilarityThreshold,
		distance:        cnf.Embedding.Distance,
	}
}

// retrieval
func (a *algorithmSimhash) getCandidateRounds(round *RoundMessage) ([]*ChatRound, error) {

	// 获取原始索引
	originalRounds, err := a.getOriginalRounds(round)
	if err != nil {
		return nil, err
	}

	// 多探针扩大搜索范围
	var rangeRounds []*ChatRound
	if a.encoder.masks16Bits != nil {
		index := a.encoder.uint16BitFlip(a.encoder.splitUint64ForUint16(round.Simhash))
		keys := convertUint16ToString(index)
		rangeRounds, err = a.getRoundsByKeys(keys)
		if err != nil {
			return nil, err
		}
	}

	rounds := make([]*ChatRound, 0, len(originalRounds)+len(rangeRounds))
	rounds = append(rounds, originalRounds...)
	rounds = append(rounds, rangeRounds...)

	return rounds, nil
}

func (a *algorithmSimhash) SimilarTextSearch(round *RoundMessage) (bool, string, error) {

	round.Simhash = a.encoder.vectorToSimhash64(round.EmbeddingVector)
	rounds, err := a.getCandidateRounds(round)
	if err != nil {
		return false, "", err
	}

	// match
	var messages []*RoundMessage

	for i := 0; i < len(rounds); i++ {
		if rounds[i] == nil {
			continue
		}

		for j := 0; j < len(rounds[i].Messages); j++ {
			msg := rounds[i].Messages[j]
			if msg.Question == round.Question {
				answer, err := a.cache.Get(msg.AnswerID)
				if err != nil {
					return true, "", err
				}
				round.AnswerID = msg.AnswerID
				return true, answer, nil

			}
			if a.encoder.calculateHammingDistanceFor64bits(round.Simhash, msg.Simhash) <= a.distance {
				messages = append(messages, msg)
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
		return false, "", nil
	}

	answer, err := a.cache.Get(messages[index].AnswerID)
	if err != nil {
		return false, "", err
	}
	round.AnswerID = messages[index].AnswerID

	return false, answer, nil
}

func (a *algorithmSimhash) InsertText(round *RoundMessage) error {

	rounds, err := a.getOriginalRounds(round)
	if err != nil {
		return err
	}

	keys := convertUint16ToString(a.encoder.splitUint64ForUint16(round.Simhash))

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

func (a *algorithmSimhash) getRoundsByKeys(keys []string) ([]*ChatRound, error) {
	// security check
	if len(keys) == 0 {
		return nil, fmt.Errorf("failed to search index")
	}

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

func (a *algorithmSimhash) getOriginalRounds(round *RoundMessage) ([]*ChatRound, error) {
	keys := convertUint16ToString(a.encoder.splitUint64ForUint16(round.Simhash))
	return a.getRoundsByKeys(keys)
}
