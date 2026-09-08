package chat_round

import (
	index_algorithm "ai-chat-service/chat-server/chat-round/index-algorithm"
	"fmt"
	"os"
	"sync"
)

var (
	once  sync.Once
	index index_algorithm.VectorIndex
)

func GetVectorIndex() (index_algorithm.VectorIndex, error) {

	if index == nil {
		return nil, fmt.Errorf("vector index is not initialized")
	}

	return index, nil
}

func InitVectorIndex(path string, dimensions int) error {

	temp, err := index_algorithm.LoadHNSWIndexSnapshot(path)
	if err == nil {
		index = temp
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	index = index_algorithm.NewHNSWIndex(index_algorithm.DefaultHNSWConfig(dimensions))

	return nil
}

func SaveVectorIndex(path string) error {

	hnswIndex, ok := index.(*index_algorithm.HNSWIndex)
	if !ok || hnswIndex == nil {
		return fmt.Errorf("hnsw index is not initialized")
	}

	return hnswIndex.SaveSnapshot(path)
}
