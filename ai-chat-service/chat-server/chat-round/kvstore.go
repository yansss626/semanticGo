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

func GetClient() (RoundCache, error) {
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

func PutClient(c RoundCache) error {
	if c == nil {
		return nil
	}

	p := pkvstore.GetPool()
	if p == nil || p.KvsPool == nil {
		return fmt.Errorf("kvstore pool is not initialized")
	}

	rc, ok := c.(*roundCache)
	if !ok {
		return fmt.Errorf("unexpected rounCache type: %T", c)
	}

	return p.KvsPool.Put(rc.kvstoreClient.Conn)

}

func (c *roundCache) Set(key string, value string) error {
	return c.kvstoreClient.Set(key, value)
}

func (c *roundCache) Get(key string) (string, error) {
	return c.kvstoreClient.Get(key)
}

func (c *roundCache) Close() {

}
