package database

import (
    "github.com/redis/go-redis/v9"
)

// InitRedis просто создает клиента. 
// Мы не передаем сюда контекст, так как само создание объекта не идет в сеть.
func InitRedis() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     "redis-cache:6379",
        Password: "",
        DB:       0,
    })
}