package kvstore

import (
	pkvstore "ai-chat-service/pkg/db/kvstore"
	"fmt"
	"net"
)

type RoundCache interface {
	Get(string) (string, error)
	Set(string, string) error
	Close()
}

type roundCache struct {
	kvstoreClient *pkvstore.Client
}

func NewKvstoreCache() (RoundCache, error) {
	p := pkvstore.GetPool()
	if p == nil || p.KvsPool == nil {
		return nil, fmt.Errorf("kvstore pool is not initialized")
	}

	c, err := p.KvsPool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from kvstore pool: %w", err)
	}

	conn, ok := c.(net.Conn)
	if !ok {
		_ = p.KvsPool.Close(c)
		return nil, fmt.Errorf("unexpected kvstore connection type: %T", c)
	}

	return &roundCache{
		kvstoreClient: &pkvstore.Client{
			Conn: conn,
		},
	}, nil
}

func (c *roundCache) Set(key string, value string) error {
	err := c.kvstoreClient.Set(key, value)
	if err != nil {
		p := pkvstore.GetPool()
		_ = p.KvsPool.Close(c.kvstoreClient.Conn)
		return err
	}
	return nil
}

func (c *roundCache) Get(key string) (string, error) {
	value, err := c.kvstoreClient.Get(key)
	if err != nil {
		p := pkvstore.GetPool()
		_ = p.KvsPool.Close(c.kvstoreClient.Conn)
		return "", err
	}
	return value, nil
}

func (c *roundCache) Close() {
	p := pkvstore.GetPool()
	_ = p.KvsPool.Put(c.kvstoreClient.Conn)
}
