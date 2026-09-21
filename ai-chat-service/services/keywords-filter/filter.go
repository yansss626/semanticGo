package keywords_filter

import (
	"ai-chat-service/mrpc_generated/filter"
	"ai-chat-service/pkg/config"
	"fmt"
	"net"
	"strconv"

	mrpc "github.com/yansss626/go-mrpc/runtime"
)

type FilterClientPool struct {
	clientPool *mrpc.ClientPool
}

func (c *FilterClientPool) Get() (*filter.FilterServiceClient, func(), error) {
	client, err := c.clientPool.Get()
	if err != nil {
		return nil, nil, err
	}
	serviceClient := filter.NewFilterServiceClient(client)
	put := func() {
		c.clientPool.Put(client)
	}
	return serviceClient, put, nil
}

func (c *FilterClientPool) Release() {
	c.clientPool.Release()
}

func InitFilterClientPool(cnf *config.Config) (*FilterClientPool, error) {
	address := cnf.DependOn.Sensitive.Address
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
	return &FilterClientPool{
		clientPool: pool,
	}, nil
}
