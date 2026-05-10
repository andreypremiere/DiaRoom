package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context) (*pgxpool.Pool, error) {
	user := os.Getenv("DIARY_DB_USER")
    pass := os.Getenv("DIARY_DB_PASSWORD")
    host := os.Getenv("DIARY_DB_HOST")
    port := os.Getenv("DIARY_DB_PORT")
    dbname := os.Getenv("DIARY_DB_NAMEBASE")

    dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", 
        user, pass, host, port, dbname)

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

    poolConfig.MaxConns = 50 

    // Минимальное количество соединений, которые всегда будут открыты
    poolConfig.MinConns = 5

    // Время, после которого неиспользуемое соединение будет закрыто
    poolConfig.MaxConnIdleTime = time.Minute * 5

    // Максимальное "время жизни" соединения (защита от утечек памяти в PG)
    poolConfig.MaxConnLifetime = time.Hour * 1

    // Время ожидания свободного соединения из пула (если все заняты)
    poolConfig.HealthCheckPeriod = time.Minute * 1

    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return nil, fmt.Errorf("unable to connect to database: %w", err)
    }

    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("database ping failed: %w", err)
    }

    return pool, nil
}