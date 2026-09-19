package chat_context

import (
	"ai-chat-service/pkg/config"
	cache_key "ai-chat-service/pkg/db/cache-key"
	"encoding/json"
	"fmt"

	"github.com/yansss626/go-lidis"
)

type cacheLidis struct {
	pool *lidis.ClientPool
}

func NewContextCache(cnf *config.Config) (ContextCache, error) {
	clientPool, err := lidis.NewClientPool(
		&lidis.ConnectionPoolConfig{
			Host: cnf.Cache.IP,
			Port: cnf.Cache.Port,
		},
	)
	if err != nil {
		return nil, err
	}
	return &cacheLidis{
		pool: clientPool,
	}, nil
}

func getContextKey(key string) string {
	return cache_key.GetKey("context", key)
}

func (l *cacheLidis) GetContext(key string) (*ChatMessage, error) {
	client, release, err := l.pool.Get()
	if err != nil {
		return nil, err
	}
	defer release()

	key = getContextKey(key)
	value, err := client.HashGet(key)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	message := &ChatMessage{}
	err = json.Unmarshal([]byte(value), message)
	if err != nil {
		return nil, fmt.Errorf("invalid ChatMessage type")
	}

	return message, nil
}

func (l *cacheLidis) SetContext(key string, message *ChatMessage) error {
	client, release, err := l.pool.Get()
	if err != nil {
		return err
	}
	defer release()

	key = getContextKey(key)
	value, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = client.HashSet(key, string(value))
	return err
}

func (l *cacheLidis) DelContext(key string) error {
	client, release, err := l.pool.Get()
	if err != nil {
		return err
	}
	defer release()

	key = getContextKey(key)
	_, err = client.HashDel(key)
	return err
}
