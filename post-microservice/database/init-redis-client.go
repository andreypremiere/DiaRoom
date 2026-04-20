package database

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func InitRedisQueue() *redis.Client {	
	addr := fmt.Sprintf("%s:%s",
		os.Getenv("REDIS_QUEUE_HOST"),
		os.Getenv("REDIS_QUEUE_POST"),
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // по умолчанию пусто
		DB:       0,  // в отдельном контейнере можно использовать 0
	})

	// Проверяем соединение при старте
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("Не удалось подключиться к Redis Queue: %v", err))
	}

	return rdb
}