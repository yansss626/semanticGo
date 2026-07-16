package kvstore

import (
	"ai-chat-service/pkg/config"
	"ai-chat-service/pkg/log"
	"fmt"
	"net"
	"time"

	"github.com/silenceper/pool"
)

type Client struct {
	Conn net.Conn
}

type KvstorePool struct {
	KvsPool pool.Pool
}

var kvsPool *KvstorePool

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
		IdleTimeout: time.Duration(cnf.Kvstore.IdleTimeout) * time.Second,
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

func GetPool() *KvstorePool {
	return kvsPool
}

func writeFully(conn net.Conn, b []byte) error {
	totalWriten := 0
	length := len(b)

	for totalWriten < length {
		n, err := conn.Write(b[totalWriten:])
		if err != nil {
			return err
		}

		totalWriten += n
	}

	return nil
}

func buildKvstoreCommand(args []string) string {
	count := len(args)
	if count == 0 {
		return ""
	}

	var command string

	// build kvsp body
	for i := 0; i < count; i++ {
		argLen := len(args[i])
		command += fmt.Sprintf("^%d&%s", argLen, args[i])

		if i == count-1 {
			command += "\r\n"
		}
	}

	bodyLen := len(command)

	// build kvsp head
	command = fmt.Sprintf("#%d\r\n", bodyLen) + command

	return command

}

func readFully(conn net.Conn) (string, error) {
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

func (c *Client) Get(key string) (string, error) {

	command := buildKvstoreCommand([]string{"HGET", key})
	if command == "" {
		return "", nil
	}

	err := writeFully(c.Conn, []byte(command))
	if err != nil {
		return "", err
	}

	value, err := readFully(c.Conn)
	if err != nil {
		return "", err
	}

	return value, nil

}

func (c *Client) Set(key string, value string) error {
	command := buildKvstoreCommand([]string{"HSET", key, value})
	if command == "" {
		return nil
	}

	err := writeFully(c.Conn, []byte(command))
	if err != nil {
		return err
	}
	_, err = readFully(c.Conn)
	if err != nil {
		return err
	}

	return nil
}
