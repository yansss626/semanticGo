package reprise

import (
	"ai-chat-service/mrpc_generated/semantic"
	"ai-chat-service/pkg/config"
	"fmt"
	"net"
	"strconv"

	mrpc "github.com/yansss626/go-mrpc/runtime"
)

type RepriseClientPool struct {
	clientPool *mrpc.ClientPool
}

func (c *RepriseClientPool) Get() (*semantic.SemanticServiceClient, func(), error) {
	client, err := c.clientPool.Get()
	if err != nil {
		return nil, nil, err
	}
	serviceClient := semantic.NewSemanticServiceClient(client)
	put := func() {
		c.clientPool.Put(client)
	}
	return serviceClient, put, nil
}

func (c *RepriseClientPool) Release() {
	c.clientPool.Release()
}

func InitRepriseClientPool(cnf *config.Config) (*RepriseClientPool, error) {
	address := cnf.DependOn.Semantic.Address
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		fmt.Println("Error parsing address:", err)
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println("Error converting port to int:", err)
		return nil, err
	}

	pool, err := mrpc.NewClientPool(&mrpc.ClientPoolConfig{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, err
	}
	return &RepriseClientPool{
		clientPool: pool,
	}, nil
}
