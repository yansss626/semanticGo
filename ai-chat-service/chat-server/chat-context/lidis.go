package chat_context

import (
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/db/cache_lidis"
	"encoding/json"
	"fmt"

	"github.com/yansss626/go-lidis"
)

type lidisCache struct {
	lidisClient *lidis.Client
}

func NewLidisCache(cnf *config.Config) (ContextCache, error) {

	client, err := lidis.NewClient(
		&lidis.ConnectionPoolConfig{
			Host: cnf.Kvstore.Host,
			Port: cnf.Kvstore.Port,
		},
	)
	if err != nil {
		return nil, err
	}
	return &lidisCache{
		lidisClient: client,
	}, nil
}

func (l *lidisCache) GetContext(key string) (*ChatMessage, error) {
	key = cache_lidis.GetKey(key)
	value, err := l.lidisClient.HashGet(key)
	if err != nil {
		return nil, err
	}
	message := &ChatMessage{}
	err = json.Unmarshal([]byte(value), message)
	if err != nil {
		return nil, fmt.Errorf("invalid ChatMessage type")
	}

	return message, nil
}

func (l *lidisCache) SetContext(key string, message *ChatMessage) error {
	key = cache_lidis.GetKey(key)
	value, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = l.lidisClient.HashSet(key, string(value))
	return err
}

func (l *lidisCache) DelContext(key string) error {
	key = cache_lidis.GetKey(key)
	_, err := l.lidisClient.HashDel(key)
	return err
}
