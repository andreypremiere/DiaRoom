package repositories

import (
	"context"
	"time"
	"user-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// UserRepositoryInter определяет интерфейс для работы с данными пользователей в Postgres и Redis
type UserRepositoryInter interface {
	// AddUser сохраняет нового пользователя в базу данных Postgres
	AddUser(ctx context.Context, id uuid.UUID, numberPhone string, roomId string) error
	
	// DeleteUserById удаляет пользователя из базы данных по его идентификатору
	DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error
	
	// AddCodeWithTimeout сохраняет временный код подтверждения в Redis на 2 минуты
	AddCodeWithTimeout(ctx context.Context, userId uuid.UUID, code string) error
	
	// GetValueByKey получает значение из Redis по идентификатору пользователя
	GetValueByKey(ctx context.Context, userId uuid.UUID) (string, error)
	
	// FindUserByPhoneOrRoomId выполняет поиск пользователя по телефону или ID комнаты
	FindUserByPhoneOrRoomId(ctx context.Context, value string) (error, *models.BaseUser)
}

// userRepository реализует методы доступа к данным через SQL и NoSQL хранилища
type userRepository struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

// AddUser выполняет вставку записи о пользователе в таблицу users
func (ur *userRepository) AddUser(ctx context.Context, id uuid.UUID, numberPhone string, roomId string) error {
	sqlInsert := "INSERT INTO users (id, phone, room_name_id) VALUES ($1, $2, $3)"
	// Выполнение SQL запроса с привязкой параметров
	_, err := ur.db.Exec(ctx, sqlInsert, id, numberPhone, roomId)
	return err
}

// DeleteUserById удаляет запись пользователя (используется для очистки при ошибках)
func (ur *userRepository) DeleteUserById(criticalCtx context.Context, userId uuid.UUID) error {
	sqlRequest := `DELETE FROM users WHERE id = $1`
	// Выполнение удаления по первичному ключу
	_, err := ur.db.Exec(criticalCtx, sqlRequest, userId)
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

// FindUserByPhoneOrRoomId ищет пользователя по одному из двух уникальных полей
func (ur *userRepository) FindUserByPhoneOrRoomId(ctx context.Context, value string) (error, *models.BaseUser) {
	// Поиск по телефону ИЛИ по room_name_id через один параметр
	query := "SELECT id, phone, room_name_id FROM users WHERE phone = $1 OR room_name_id = $1"

	foundUser := models.BaseUser{}

	// Сканирование результата в структуру BaseUser
	err := ur.db.QueryRow(ctx, query, value).Scan(&foundUser.Id, &foundUser.NumberPhone, &foundUser.RoomId)
	if err != nil {
		return err, nil
	}

	return nil, &foundUser
}

// NewUserRepository создает экземпляр репозитория с пулом Postgres и клиентом Redis
func NewUserRepository(dbConnection *pgxpool.Pool, rdb *redis.Client) *userRepository {
	return &userRepository{db: dbConnection, rdb: rdb}
}