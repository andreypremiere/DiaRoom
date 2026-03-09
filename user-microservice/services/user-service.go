package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"user-microservice/models"
	"user-microservice/repositories"
	"user-microservice/utils"

	"github.com/andreypremiere/jwtmanager"
	"github.com/google/uuid"
)

// UserServiceInter определяет интерфейс управления пользователями и их авторизацией
type UserServiceInter interface {
	// AddUser регистрирует пользователя и инициирует создание комнаты
	AddUser(ctx context.Context, user *models.RegisterUser) (uuid.UUID, error)
	// VerifyCode проверяет код из Redis и возвращает JWT токен
	VerifyCode(ctx context.Context, verifyUser models.VerifyUserById) (string, error)
	// LoginUser инициирует процесс входа через отправку кода
	LoginUser(ctx context.Context, value string) (error, uuid.UUID)
}

// userService реализует бизнес-логику с использованием репозитория и SMS-провайдера
type userService struct {
	userRepo    repositories.UserRepositoryInter
	smsProvider utils.SmsProvider
}

// AddUser выполняет комплексный процесс регистрации пользователя и комнаты
func (us userService) AddUser(ctx context.Context, user *models.RegisterUser) (uuid.UUID, error) {
	newUUID := uuid.New()
	if user.Id != uuid.Nil {
		newUUID = user.Id
	}

	// Валидация обязательных полей перед началом транзакции
	if user.RoomId == "" {
		return uuid.Nil, errors.New("Room Id cannot be empty")
	}
	if user.NumberPhone == "" {
		return uuid.Nil, errors.New("NumberPhone cannot be empty")
	}

	// Сохранение базовой записи пользователя в Postgres
	err := us.userRepo.AddUser(ctx, newUUID, user.NumberPhone, user.RoomId)
	if err != nil {
		fmt.Println("Ошибка добавления пользователя в базу данных", err)
		return uuid.Nil, err
	}

	// Ограничение времени на сетевой запрос к соседнему микросервису
	reqCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()

	// Подготовка данных для создания профиля комнаты
	url := "http://room-microservice:81/newRoom"
	roomCreating := models.RoomCreating{UserId: newUUID, RoomName: user.RoomName, RoomNameId: user.RoomId}
	body, err := json.Marshal(roomCreating)
	if err != nil {
		return uuid.Nil, err
	}

	// Выполнение ме сервисного запроса
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка при обращении к микросервису rooms", err.Error())
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	// Компенсирующая транзакция: удаление юзера при ошибке создания комнаты
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Внешний сервис вернул ошибку: ", resp.StatusCode)
		err := us.DeleteUserById(ctx, newUUID)
		if err != nil {
			fmt.Println("Ошибка удаления пользователя при ошибке его создания. БАГ.", err.Error())
			return uuid.Nil, err
		}
		return uuid.Nil, errors.New("Микросервис вернул плохой статус код")
	} 

	// Генерация и отправка проверочного кода через SMS
	err = us.GenerateAndSendCode(ctx, newUUID, user.NumberPhone)
	if err != nil {
		return uuid.Nil, err
	}

	return newUUID, nil
}

// DeleteUserById удаляет пользователя с игнорированием отмены родительского контекста
func (us *userService) DeleteUserById(ctx context.Context, userId uuid.UUID) error {
	// Использование context.WithoutCancel гарантирует завершение удаления
	criticalCtx := context.WithoutCancel(ctx)
	err := us.userRepo.DeleteUserById(criticalCtx, userId)
	return err
}

// AddCodeWithTimeout сохраняет временный секретный код в хранилище
func (us *userService) AddCodeWithTimeout(ctx context.Context, userId uuid.UUID, code string) error {
	err := us.userRepo.AddCodeWithTimeout(ctx, userId, code)
	return err
}

// VerifyCode проверяет код и генерирует JWT при успехе
func (us *userService) VerifyCode(ctx context.Context, verifyUser models.VerifyUserById) (string, error) {
	// Получение эталонного кода из Redis
	gotCode, err := us.userRepo.GetValueByKey(ctx, verifyUser.UserId)
	if err != nil {
		return "", errors.Join(errors.New("Что-то пошло не так во время получения кода"), err)
	}

	// Сравнение кодов
	if gotCode != verifyUser.Code {
		return "", errors.New("Введенный код не совпадает с полученным")
	}

	// Получение ID комнаты для включения в Payload токена
	roomId, err := GetRoomIdByUserId(verifyUser.UserId)
	if err != nil {
		return "", errors.Join(errors.New("Не удалось получить комнату"), err)
	}

	secretJwt := os.Getenv("JWT_SECRET")

	// Формирование JWT для авторизации в Gateway
	jwtmanager := jwtmanager.NewJWTManager(secretJwt, 30*time.Minute)
	token, err := jwtmanager.Generate(verifyUser.UserId.String(), roomId.String())
	if err != nil {
		return "", errors.Join(errors.New("Ошибка при генерации токена"), err)
	}

	return token, nil
}

// LoginUser ищет пользователя и отправляет ему новый код для входа
func (us *userService) LoginUser(ctx context.Context, value string) (error, uuid.UUID) {
	err, user := us.FindUserByPhoneOrRoomId(ctx, value)
	if err != nil {
		return err, uuid.Nil
	}

	err = us.GenerateAndSendCode(ctx, user.Id, user.NumberPhone)
	if err != nil {
		return err, uuid.Nil
	}

	return nil, user.Id
}

// FindUserByPhoneOrRoomId вспомогательный метод поиска пользователя
func (us *userService) FindUserByPhoneOrRoomId(ctx context.Context, value string) (error, *models.BaseUser) {
	err, user := us.userRepo.FindUserByPhoneOrRoomId(ctx, value)
	if err != nil {
		return err, nil
	}
	return nil, user
}

// GenerateAndSendCode отвечает за полный цикл работы с OTP (генерация, хранение, отправка)
func (us *userService) GenerateAndSendCode(ctx context.Context, userId uuid.UUID, numberPhone string) error {
	code, err := utils.GenerateCode()
	if err != nil {
		return err
	}

	// Сохранение кода в кэш
	err = us.AddCodeWithTimeout(ctx, userId, code)
	if err != nil {
		return err
	}

	// Физическая отправка через провайдера
	err = us.smsProvider.SendCode(numberPhone, code)
	if err != nil {
		return  err
	}

	return nil
}

// NewUserService создает экземпляр сервиса с необходимыми зависимостями
func NewUserService(userRepo repositories.UserRepositoryInter, smsProvider utils.SmsProvider) *userService {
	return &userService{userRepo: userRepo, smsProvider: smsProvider}
}