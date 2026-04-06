package repositories

import (
	"context"
	"fmt"
	"time"
	"user-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// UserRepositoryInter определяет интерфейс для работы с данными пользователей в Postgres и Redis
type UserRepositoryInter interface {
	// AddUser сохраняет нового пользователя в базу данных Postgres
	AddUser(ctx context.Context, id uuid.UUID, email string, passwordHash string) error

	// DeleteUserById удаляет пользователя из базы данных по его идентификатору
	DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error

	// AddCodeWithTimeout сохраняет временный код подтверждения в Redis на 2 минуты
	AddCodeWithTimeout(ctx context.Context, userId uuid.UUID, code string) error

	// GetValueByKey получает значение из Redis по идентификатору пользователя
	GetValueByKey(ctx context.Context, userId uuid.UUID) (string, error)

	GetEmailByUserId(ctx context.Context, id uuid.UUID) (string, error)

	GetUserByEmail(ctx context.Context, email string) (*models.CheckUser, error)

	SetUserVerified(ctx context.Context, userID uuid.UUID) error

	DeleteRefreshToken(ctx context.Context, token string) error

	GetSessionByToken(ctx context.Context, token string) (*models.Session, error)

	UpdateRefreshToken(ctx context.Context, oldToken, newToken string, expiresAt time.Time) error

	AddRefreshToken(ctx context.Context, userID uuid.UUID, refreshToken string, userAgent string, expiresAt time.Time) error
}

// userRepository реализует методы доступа к данным через SQL и NoSQL хранилища
type userRepository struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

// AddRefreshToken сохраняет новую сессию пользователя в таблицу sessions
func (ur *userRepository) AddRefreshToken(ctx context.Context, userID uuid.UUID, refreshToken string, userAgent string, expiresAt time.Time) error {
	sqlInsert := `
        INSERT INTO sessions (user_id, refresh_token, user_agent, expires_at) 
        VALUES ($1, $2, $3, $4)`

	_, err := ur.db.Exec(ctx, sqlInsert, userID, refreshToken, userAgent, expiresAt)
	if err != nil {
		return err
	}
	return nil
}

// AddUser выполняет вставку записи о пользователе в таблицу users
func (ur *userRepository) AddUser(ctx context.Context, id uuid.UUID, email string, passwordHash string) error {
	sqlInsert := "INSERT INTO users (id, email, hash_password) VALUES ($1, $2, $3)"
	// Выполнение SQL запроса с привязкой параметров
	_, err := ur.db.Exec(ctx, sqlInsert, id, email, passwordHash)
	return err
}

func (ur *userRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := ur.db.Exec(ctx, "DELETE FROM sessions WHERE refresh_token = $1", token)
	return err
}

func (ur *userRepository) UpdateRefreshToken(ctx context.Context, oldToken, newToken string, expiresAt time.Time) error {
	sql := `
        UPDATE sessions 
        SET refresh_token = $1, expires_at = $2
        WHERE refresh_token = $3`

	res, err := ur.db.Exec(ctx, sql, newToken, expiresAt, oldToken)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("Сессия не была обновлена")
	}

	return nil
}

func (ur *userRepository) GetSessionByToken(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session

	query := `
        SELECT id, user_id, refresh_token, expires_at 
        FROM sessions 
        WHERE refresh_token = $1 
        LIMIT 1`

	err := ur.db.QueryRow(ctx, query, token).Scan(
		&session.Id,
		&session.UserId,
		&session.RefreshToken,
		&session.ExpiresAt,
	)

	if err != nil {
		// Если строк не найдено
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("сессия не найдена")
		}
		// Системная ошибка (проблема с БД)
		return nil, fmt.Errorf("ошибка при получении сессии: %w", err)
	}

	return &session, nil
}

// DeleteUserById удаляет запись пользователя (используется для очистки при ошибках)
func (ur *userRepository) DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error {
	sqlRequest := `DELETE FROM users WHERE id = $1`
	// Выполнение удаления по первичному ключу
	_, err := ur.db.Exec(criticalCtx, sqlRequest, userId)
	return err
}

// GetEmailByUserId возвращает email пользователя по его UUID
func (ur *userRepository) GetEmailByUserId(ctx context.Context, id uuid.UUID) (string, error) {
	var email string

	// Пишем запрос. Нам нужно только поле email
	sqlSelect := "SELECT email FROM users WHERE id = $1"

	// Используем QueryRow, так как ID уникален и мы ждем ровно одну строку
	err := ur.db.QueryRow(ctx, sqlSelect, id).Scan(&email)

	if err != nil {
		// Если пользователь не найден, pgx вернет pgx.ErrNoRows
		if err.Error() == "no rows in result set" {
			return "", fmt.Errorf("пользователь с id %s не найден", id)
		}
		return "", fmt.Errorf("ошибка при получении email: %w", err)
	}

	return email, nil
}

// GetUserByEmail ищет пользователя и возвращает структуру CheckUser для проверки пароля
func (ur *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.CheckUser, error) {
	// Инициализируем структуру
	var user models.CheckUser

	// Пишем запрос, перечисляя все нужные поля
	// Важно: порядок полей в SELECT должен строго совпадать с порядком в Scan()
	sqlSelect := `
        SELECT id, email, is_activated, hash_password 
        FROM users 
        WHERE email = $1`

	// Выполняем запрос
	err := ur.db.QueryRow(ctx, sqlSelect, email).Scan(
		&user.Id,           // Поле из вложенной BaseUser
		&user.Email,        // Поле из вложенной BaseUser
		&user.IsActivated,  // Поле из вложенной BaseUser (проверь тип string/bool)
		&user.HashPassword, // Поле из CheckUser
	)

	if err != nil {
		// Обработка случая, когда пользователь не найден
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("Пользователь с email %s не найден", email)
		}
		// Оборачиваем системную ошибку через %w
		return nil, fmt.Errorf("Ошибка при поиске пользователя по email: %w", err)
	}

	return &user, nil
}

func (ur *userRepository) SetUserVerified(ctx context.Context, userID uuid.UUID) error {
	// Обновляем только если статус еще false
	sql := "UPDATE users SET is_activated = true WHERE id = $1 AND is_activated = false"
	_, err := ur.db.Exec(ctx, sql, userID)
	return err
}

// AddCodeWithTimeout устанавливает пару ключ-значение в Redis с автоматическим удалением
func (ur *userRepository) AddCodeWithTimeout(
	ctx context.Context, userId uuid.UUID, code string) error {
	// Сохранение кода с TTL (временем жизни) 2 минуты
	err := ur.rdb.Set(ctx, userId.String(), code, 2*time.Minute).Err()
	return err
}

// GetValueByKey возвращает строку из Redis по строковому представлению UUID
func (ur *userRepository) GetValueByKey(ctx context.Context, userId uuid.UUID) (string, error) {
	result := ur.rdb.Get(ctx, userId.String())
	// Возврат строкового значения и возможной ошибки выполнения
	return result.Val(), result.Err()
}

// NewUserRepository создает экземпляр репозитория с пулом Postgres и клиентом Redis
func NewUserRepository(dbConnection *pgxpool.Pool, rdb *redis.Client) *userRepository {
	return &userRepository{db: dbConnection, rdb: rdb}
}
