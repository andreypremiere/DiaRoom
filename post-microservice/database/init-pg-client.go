package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitPool(ctx context.Context) (*pgxpool.Pool, error) {

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POST_DB_USER"),
		os.Getenv("POST_DB_PASSWORD"),
		os.Getenv("POST_DB_HOST"),
		os.Getenv("POST_DB_PORT"),
		os.Getenv("POST_DB_NAMEBASE"),
	)

	// Создаем конфиг пула
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DSN: %w", err)
	}

	// Настройки пула для Highload (как мы обсуждали для 10k юзеров)
	config.MaxConns = 25                      // Максимальное кол-во соединений
	config.MinConns = 5                       // Минимальное кол-во активных соединений
	config.MaxConnLifetime = time.Hour        // Время жизни соединения
	config.MaxConnIdleTime = 30 * time.Minute // Время простоя

	// Инициализируем пул
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Важно: проверяем реальное подключение к базе
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	fmt.Println("Пул соединений инициализирован")

	return pool, nil
}