package repositories

import (
	"account-microservice/models"
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
    return errors.New("Неизвестная ошибка в бд")
}

func (ar *AccountRepository) GetOTPCode(ctx context.Context, userId uuid.UUID) (string, error) {
    code, err := ar.redisClient.Get(ctx, userId.String()).Result()
    
    if err != nil {
        if errors.Is(err, redis.Nil) {
            return "", errors.New("code expired") 
        }
        return "", errors.New("Неизвестная ошибка получения кода OTP")
    }

    return code, nil
}

func (ar *AccountRepository) GetStatusConfigured(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isConfigured bool
	query := `SELECT is_configured FROM users WHERE id = $1`

	err := ar.poolPg.QueryRow(ctx, query, userID).Scan(&isConfigured)
	if err != nil {
		return false, ar.parsePgError(err)
	}

	return isConfigured, nil
}

func (ar *AccountRepository) GetRoomIdByUserId(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var roomID uuid.UUID
	query := `SELECT id FROM rooms WHERE user_id = $1`

	err := ar.poolPg.QueryRow(ctx, query, userID).Scan(&roomID)
	if err != nil {
		return uuid.Nil, ar.parsePgError(err)
	}

	return roomID, nil
}

func (ar *AccountRepository) UpdateRefreshToken(
    ctx context.Context, 
    oldToken string, 
    newToken string, 
    newExpiresAt time.Time,
) error {
    query := `
        UPDATE sessions 
        SET refresh_token = $1, expires_at = $2 
        WHERE refresh_token = $3
    `

    result, err := ar.poolPg.Exec(ctx, query, newToken, newExpiresAt, oldToken)
    if err != nil {
        return errors.New("Ошибка в бд")
    }

    if result.RowsAffected() == 0 {
        return errors.New("сессия для обновления не найдена")
    }

    return nil
}

func (ar *AccountRepository) GetUserByEmail(ctx context.Context, email string) (*models.BaseUser, error) {
    user := &models.BaseUser{}
    
    query := `SELECT id, email, hash_password, is_activated FROM users WHERE email = $1`

    err := ar.poolPg.QueryRow(ctx, query, email).Scan(
        &user.ID, 
        &user.Email, 
        &user.PasswordHash, 
        &user.IsActivated,
    )

    if err != nil {
        return nil, ar.parsePgError(err)
    }

    return user, nil
}

func (ar *AccountRepository) VerifyAndCreateSession(
    ctx context.Context, 
    userID uuid.UUID, 
    refreshToken string, 
    deviceInfo string, 
    expiresAt time.Time,
) error {
    tx, err := ar.poolPg.Begin(ctx)
    if err != nil {
        return fmt.Errorf("ошибка начала транзакции: %w", err)
    }

    defer tx.Rollback(ctx)

    const updateStatusQuery = `UPDATE users SET is_activated = true WHERE id = $1`
    _, err = tx.Exec(ctx, updateStatusQuery, userID)
    if err != nil {
        return ar.parsePgError(err) 
    }

    const addTokenQuery = `
        INSERT INTO sessions (user_id, refresh_token, user_agent, expires_at)
        VALUES ($1, $2, $3, $4)
    `
    _, err = tx.Exec(ctx, addTokenQuery, userID, refreshToken, deviceInfo, expiresAt)
    if err != nil {
        return ar.parsePgError(err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("ошибка фиксации транзакции: %w", err)
    }

    return nil
}

func (ar *AccountRepository) GetSessionByToken(ctx context.Context, token string) (*models.SessionWithRoomId, error) {
    session := &models.SessionWithRoomId{}

    query := `
        SELECT 
            rt.user_id, 
            rt.refresh_token, 
            rt.user_agent, 
            rt.expires_at,
            r.id as room_id
        FROM sessions rt
        JOIN rooms r ON rt.user_id = r.user_id
        WHERE rt.refresh_token = $1
    `

    err := ar.poolPg.QueryRow(ctx, query, token).Scan(
        &session.UserId,
        &session.RefreshToken,
        &session.UserAgent,
        &session.ExpiresAt,
        &session.RoomId,
    )

    if err != nil {
        return nil, errors.New("Токен не найден или его нет")
    }

    return session, nil
}

func (ar *AccountRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE refresh_token = $1`

	_, err := ar.poolPg.Exec(ctx, query, token)
	if err != nil {
		return errors.New("Ошибка во время удаления токена")
	}

	return nil
}

func (ar *AccountRepository) GetUserEmailByID(ctx context.Context, userID uuid.UUID) (*models.EmailUser, error) {
	var user models.EmailUser

	query := `SELECT id, email FROM users WHERE id = $1`
	err := ar.poolPg.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Email)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, errors.New("нет такого пользователя")
		}
		return nil, err
	}
	return &user, nil
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