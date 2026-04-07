package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AccountRepository struct {
	poolPg *pgxpool.Pool
	redisClient *redis.Client
}

func (ar *AccountRepository) AddCodeWithTimeout(
	ctx context.Context, userId uuid.UUID, code string) error {
	// Сохранение кода с TTL (временем жизни) 2 минуты
	err := ar.redisClient.Set(ctx, userId.String(), code, 2*time.Minute).Err()
	return err
}

func (ar *AccountRepository) NewAccount(ctx context.Context, email string, userID, roomID uuid.UUID, roomUniqueId, roomName, hashPassword string) error {
	// Начинаем транзакцию
	tx, err := ar.poolPg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("Ошибка старта транзакции: %w", err)
	}

	// Гарантируем откат, если функция завершится до Commit
	defer tx.Rollback(ctx)

	// 2. Вставляем пользователя
	userQuery := `
		INSERT INTO users (id, email, hash_password) 
		VALUES ($1, $2, $3)`
	
	_, err = tx.Exec(ctx, userQuery, userID, email, hashPassword)
	if err != nil {
		return ar.parsePgError(err)
	}

	// 3. Вставляем комнату
	roomQuery := `
		INSERT INTO rooms (id, user_id, room_unique_id, room_name) 
		VALUES ($1, $2, $3, $4)`
	
	_, err = tx.Exec(ctx, roomQuery, roomID, userID, roomUniqueId, roomName)
	if err != nil {
		return ar.parsePgError(err)
	}

	// 4. Подтверждаем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Ошибка завершения транзакции: %w", err)
	}

	return nil
}

func (ar *AccountRepository) parsePgError(err error) error {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        fmt.Printf("[DB DEBUG] Code: %s | Message: %s | Constraint: %s\n", 
            pgErr.Code, pgErr.Message, pgErr.ConstraintName)

        switch pgErr.Code {
        case "23505": // Unique Violation
            switch pgErr.ConstraintName {
            case "users_email_key":
                return errors.New("Пользователь уже существует") // Email уже занят
            case "rooms_room_unique_id_key":
                return errors.New("@roomid уже занято") // Никнейм @user уже занят
            }

        case "22001": // String Data Right Truncation
            // Если в ошибке есть имя колонки, можно уточнить
            return errors.New("Слишком длинный текст") // Слишком длинный текст

        case "23502": // Not Null Violation
            return errors.New("Пропущено обязательное поле") // Пропущено обязательное поле

        case "23503": // Foreign Key Violation
            return errors.New("Ошибка связи данных") // Ошибка связи данных

		default:
            // Если код ошибки Postgres нам не знаком
            return errors.New("Произошла системная ошибка базы данных")
        }
    }
    return errors.New("Неизвестная ошибка при создании аккаунта")
}

func NewAccountRepository(
	poolPg *pgxpool.Pool,
	redisClient *redis.Client,
) *AccountRepository {
	return &AccountRepository{
		poolPg: poolPg,
		redisClient: redisClient,
	}
}