package repositories

import (
	"context"
	"time"

	// "fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type UserRepositoryInter interface {
	AddUser(ctx context.Context, id uuid.UUID, numberPhone string) error
	DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error
	AddCodeWithTimeout(ctx context.Context, userId uuid.UUID, code string) error
}

type userRepository struct {
	db *pgxpool.Pool 
	rdb *redis.Client
}

func (ur *userRepository) AddUser(ctx context.Context, id uuid.UUID, numberPhone string) error {
	sqlInsert := "INSERT INTO users (id, phone) VALUES ($1, $2)"
	_, err := ur.db.Exec(ctx, sqlInsert, id, numberPhone)
	return err
}

func (ur *userRepository) DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error {
	sqlRequest := `DELETE FROM users WHERE id = $1`
	_, err := ur.db.Exec(criticalCtx, sqlRequest, userId)
	return err
}	

func (ur *userRepository) AddCodeWithTimeout(
	ctx context.Context, userId uuid.UUID, code string) error {
	err := ur.rdb.Set(ctx, userId.String(), code, 2*time.Minute).Err()
	return err
}

func NewUserRepository(dbConnection *pgxpool.Pool, rdb *redis.Client) *userRepository {
	return &userRepository{db: dbConnection, rdb: rdb}
}