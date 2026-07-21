package kvstore

import (
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/silenceper/pool"
)

type Client struct {
	Conn   net.Conn
	reader *bufio.Reader
}

type KvstorePool struct {
	KvsPool pool.Pool
}

var kvsPool *KvstorePool

func GetPool() *KvstorePool {
	return kvsPool
}

func InitKvstorePool(cnf *config.Config) error {
	addr := fmt.Sprintf("%s:%d", cnf.Kvstore.Host, cnf.Kvstore.Port)

	factory := func() (interface{}, error) { return net.Dial("tcp", addr) }

	close := func(v interface{}) error { return v.(net.Conn).Close() }

	poolConfig := &pool.Config{
		InitialCap:  cnf.Kvstore.InitialCap,
		MaxIdle:     cnf.Kvstore.MaxIdle,
		MaxCap:      cnf.Kvstore.MaxCap,
		Factory:     factory,
		Close:       close,
		IdleTimeout: time.Duration(cnf.Kvstore.IdleTimeout) * time.Minute,
	}
	pool, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		log.Error(err)
		return err
	}

	kvsPool = &KvstorePool{
		KvsPool: pool,
	}
	return nil
}

func NewClient(conn net.Conn) *Client {
	return &Client{
		Conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

func (c *Client) Get(key string) (string, error) {

	err := c.writeRequest([]string{"HGET", key})
	if err != nil {
		return "", err
	}

	Reply, err := c.readReply()
	if err != nil {
		return "", err
	}

	if Reply.Type != StringReply {
		if Reply.Type == ErrorReply {
			return "", fmt.Errorf("%s", Reply.Bytes)
		}
		return "", fmt.Errorf("reply type mistach: expected string, got %c", Reply.Type)
	}

	if Reply.IsNull {
		return "", nil

	}

	return string(Reply.Bytes), nil
}

func (c *Client) Set(key, value string) (bool, error) {

	err := c.writeRequest([]string{"HSET", key, value})
	if err != nil {
		return false, err
	}

	Reply, err := c.readReply()
	if err != nil {
		return false, err
	}

	if Reply.Type != StatusReply {
		if Reply.Type == ErrorReply {
			return false, fmt.Errorf("%s", Reply.Bytes)
		}

		if !Reply.IsNull {
			return false, nil
		}
		return false, fmt.Errorf("reply type mistach: expected string, got %c", Reply.Type)
	}

	return true, nil

}

func (c *Client) Mod(key, value string) (bool, error) {
	err := c.writeRequest([]string{"HMOD", key, value})
	if err != nil {
		return false, err
	}

	Reply, err := c.readReply()
	if err != nil {
		return false, err
	}

	if Reply.Type != StatusReply {
		if Reply.Type == ErrorReply {
			return false, fmt.Errorf("%s", Reply.Bytes)
		}

		if Reply.Type == IntegerReply {
			return false, nil
		}
		return false, fmt.Errorf("reply type mistach: expected string, got %c", Reply.Type)
	}

	return true, nil

}
