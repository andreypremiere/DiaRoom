package database

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient инициализирует подключение к Redis
func NewRedisClient(ctx context.Context) (*redis.Client, error) {
	host := os.Getenv("REDIS_DIARY_HOST")
	if host == "" {
		return nil, fmt.Errorf("REDIS_HOST is not set")
	}

	port := os.Getenv("REDIS_DIARY_PORT")
	if port == "" {
		return nil, fmt.Errorf("REDIS_PORT is not set")
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}
