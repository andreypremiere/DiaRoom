package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitPool(ctx context.Context) (*pgxpool.Pool, error) {
	// Собираем DSN из переменных окружения Docker
	// Пример: postgres://user:password@localhost:5432/dbname
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"postgres",
		os.Getenv("DB_PASSWORD"),
		"postgresql-posts",
		"5432",
		"db_posts",
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

	return pool, nil
}