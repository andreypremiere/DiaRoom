package repositories

import (
	"context"
	// "fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepositoryInter interface {
	AddUser(ctx context.Context, id uuid.UUID, numberPhone string) error
}

type userRepository struct {
	db *pgxpool.Pool // Измениться потом тип данных и добавить
}

func (ur *userRepository) AddUser(ctx context.Context, id uuid.UUID, numberPhone string) error {
		sqlInsert := "INSERT INTO users (id, phone) VALUES ($1, $2)"
		_, err := ur.db.Exec(ctx, sqlInsert, id, numberPhone)
		return err
	}

func NewUserRepository(dbConnection *pgxpool.Pool) *userRepository {
	return &userRepository{db: dbConnection}
}