package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultDialTimeout = 5 * time.Second
	defaultReadTimeout = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// NewRedisClient returns a configured go-redis client and validates the connection with PING.
func NewRedisClient(addr, password string) (*redis.Client, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, errors.New("redis: addr is empty")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DialTimeout:  defaultDialTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

