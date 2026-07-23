package chat_round

import (
	"ai-chat-service/pkg/config"
	pkvstore "ai-chat-service/pkg/db/kvstore"
	"fmt"
	"net"
)

type RoundCache interface {
	Get(string) (string, error)
	Set(string, string) (bool, error)
	Mod(string, string) (bool, error)
	Close()
}

type roundCache struct {
	kvstoreClient *pkvstore.Client
	invalid       bool
}

func NewKvstoreCache() (RoundCache, error) {
	p := pkvstore.GetPool()
	if p == nil || p.KvsPool == nil {
		err := pkvstore.InitKvstorePool(config.GetConfig())
		if err != nil {
			return nil, err
		}
		p = pkvstore.GetPool()
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
		kvstoreClient: pkvstore.NewClient(conn),
		invalid:       false,
	}, nil
}

func (c *roundCache) invalidate() {
	if c.invalid {
		return
	}
	p := pkvstore.GetPool()
	_ = p.KvsPool.Close(c.kvstoreClient.Conn)
	c.invalid = true
}

func (c *roundCache) Set(key, value string) (bool, error) {
	isSuccess, err := c.kvstoreClient.Set(key, value)
	if err != nil {
		c.invalidate()
		return false, err
	}
	return isSuccess, nil
}

func (c *roundCache) Get(key string) (string, error) {
	value, err := c.kvstoreClient.Get(key)
	if err != nil {
		c.invalidate()
		return "", err
	}
	return value, nil
}

func (c *roundCache) Mod(key, value string) (bool, error) {
	isSuccess, err := c.kvstoreClient.Mod(key, value)
	if err != nil {
		p := pkvstore.GetPool()
		_ = p.KvsPool.Close(c.kvstoreClient.Conn)
		return false, err
	}
	return isSuccess, nil
}

func (c *roundCache) Close() {
	if c.invalid {
		return
	}
	p := pkvstore.GetPool()
	_ = p.KvsPool.Put(c.kvstoreClient.Conn)
}
