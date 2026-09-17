package ai_chat_service

import (
	"ai-chat-backend/mrpc_generated/chat"
	"ai-chat-backend/pkg/config"
	"fmt"
	"net"
	"strconv"

	mrpc "github.com/yansss/mrpc/runtime"
)

type ChatServiceClientPool struct {
	clientPool *mrpc.ClientPool
}

func (c *ChatServiceClientPool) Get() (*chat.ChatServiceClient, func(), error) {
	client, err := c.clientPool.Get()
	if err != nil {
		return nil, nil, err
	}
	serviceClient := chat.NewChatServiceClient(client)
	put := func() {
		c.clientPool.Put(client)
	}
	return serviceClient, put, nil
}

func (c *ChatServiceClientPool) Release() {
	c.clientPool.Release()
}

func InitFilterClientPool(cnf *config.Config) (*ChatServiceClientPool, error) {
	address := cnf.DependOn.AiChatService.Address
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
	return &ChatServiceClientPool{
		clientPool: pool,
	}, nil
}
