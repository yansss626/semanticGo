package chat_round

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"ai-chat-service/pkg/config"
	"sync"
)

var (
	once  sync.Once
	index index_algorithm.VectorIndex
)

func GetVectorIndex() index_algorithm.VectorIndex {

	once.Do(func() {
		cnf := config.GetConfig()

		hnswCnf := index_algorithm.DefaultHNSWConfig(cnf.Embedding.VectorDimensions)
		index = index_algorithm.NewHNSWIndex(hnswCnf)
	})

	return index
}
