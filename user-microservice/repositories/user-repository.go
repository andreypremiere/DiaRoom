package repositories

import (
	"context"
	"time"
	"user-microservice/models"

	// "fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type UserRepositoryInter interface {
	AddUser(ctx context.Context, id uuid.UUID, numberPhone string, roomId string) error
	DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error
	AddCodeWithTimeout(ctx context.Context, userId uuid.UUID, code string) error
	GetValueByKey(ctx context.Context, userId uuid.UUID) (string, error)
	FindUserByPhoneOrRoomId(ctx context.Context, value string) (error, *models.BaseUser)
}

type userRepository struct {
	db *pgxpool.Pool 
	rdb *redis.Client
}

func (ur *userRepository) AddUser(ctx context.Context, id uuid.UUID, numberPhone string, roomId string) error {
	sqlInsert := "INSERT INTO users (id, phone, room_name_id) VALUES ($1, $2, $3)"
	_, err := ur.db.Exec(ctx, sqlInsert, id, numberPhone, roomId)
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

func (ur *userRepository) GetValueByKey(ctx context.Context, userId uuid.UUID) (string, error) {
	result := ur.rdb.Get(ctx, userId.String())
	
	return result.Val(), result.Err()
}

func (ur *userRepository) FindUserByPhoneOrRoomId(ctx context.Context, value string) (error, *models.BaseUser) {
	query := "SELECT id, phone, room_name_id FROM users WHERE phone = $1 OR room_name_id = $1"

	foundUser := models.BaseUser{}

	err := ur.db.QueryRow(ctx, query, value).Scan(&foundUser.Id, &foundUser.NumberPhone, &foundUser.RoomId)
	if err != nil {
		return err, nil
	}

	return nil, &foundUser
}

func NewUserRepository(dbConnection *pgxpool.Pool, rdb *redis.Client) *userRepository {
	return &userRepository{db: dbConnection, rdb: rdb}
}

