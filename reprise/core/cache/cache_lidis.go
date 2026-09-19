package cache

import (
	"encoding/json"
	"reprise/pkg/cache/cache_key"
	"reprise/pkg/config"

	"github.com/yansss626/go-lidis"
)

type cacheLidis struct {
	pool *lidis.ClientPool
}

func NewCache(cnf *config.Config) (CacheObject, error) {
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

func getKey(key string) string {
	return cache_key.GetKey(key)
}

func (l *cacheLidis) Set(key string, enrty *CacheEntry) error {
	client, release, err := l.pool.Get()
	if err != nil {
		return err
	}
	defer release()

	key = getKey(key)
	value, err := json.Marshal(enrty)
	if err != nil {
		return err
	}

	_, err = client.HashSet(key, string(value))
	return err
}

func (l *cacheLidis) Get(key string) (*CacheEntry, error) {
	client, release, err := l.pool.Get()
	if err != nil {
		return nil, err
	}
	defer release()

	key = getKey(key)
	value, err := client.HashGet(key)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	entry := &CacheEntry{}
	err = json.Unmarshal([]byte(value), entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (l *cacheLidis) Del(key string) error {
	client, release, err := l.pool.Get()
	if err != nil {
		return err
	}
	defer release()
	key = getKey(key)
	_, err = client.HashDel(key)
	return err
}
