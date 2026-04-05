package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"user-microservice/contracts/requests"
	"user-microservice/models"
	"user-microservice/repositories"
	"user-microservice/utils"

	"github.com/andreypremiere/jwtmanager"
	"github.com/google/uuid"
)

// UserServiceInter определяет интерфейс управления пользователями и их авторизацией
type UserServiceInter interface {
	// AddUser регистрирует пользователя и инициирует создание комнаты
	AddUser(ctx context.Context, user *requests.UserCreatingContract) (uuid.UUID, error)
	// VerifyCode проверяет код из Redis и возвращает JWT токен
	VerifyCode(ctx context.Context, verifyUser models.VerifyUserById) (string, error)
	// LoginUser инициирует процесс входа через отправку кода
	LoginUser(ctx context.Context, value string) (error, uuid.UUID)
}

// userService реализует бизнес-логику с использованием репозитория и SMS-провайдера
type userService struct {
	userRepo    repositories.UserRepositoryInter
	emailProvider *utils.MailService
	passHasher *utils.PasswordHasher
}

// AddUser выполняет комплексный процесс регистрации пользователя и комнаты
func (us userService) AddUser(ctx context.Context, user *requests.UserCreatingContract) (uuid.UUID, error) {
	newUUID := uuid.New()

	// Валидация обязательных полей перед началом транзакции
	if user.Email == "" {
		return uuid.Nil, errors.New("Поле Email оказалось пустым")
	}
	if user.Password == "" {
		return uuid.Nil, errors.New("Поле Password оказалось пустым")
	}

	// Сделать хеш пароля
	hashPassword, err := us.passHasher.HashPassword(user.Password)
	if err != nil {
		return uuid.Nil, err
	}

	// Сохранение базовой записи пользователя в Postgres
	err = us.userRepo.AddUser(ctx, newUUID, user.Email, hashPassword)
	if err != nil {
		return uuid.Nil, err
	}

	// Генерация и отправка проверочного кода через SMS
	err = us.GenerateAndSendCode(ctx, newUUID, user.Email)
	if err != nil {
		fmt.Println("Возникла ошибка во время создания или отправки кода")
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
	jwtmanager := jwtmanager.NewJWTManager(secretJwt, 540*time.Minute)
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
func (us *userService) GenerateAndSendCode(ctx context.Context, userId uuid.UUID, email string) error {
	code, err := utils.GenerateCode()
	if err != nil {
		return err
	}

	// Сохранение кода в базу редис
	err = us.AddCodeWithTimeout(ctx, userId, code)
	if err != nil {
		return err
	}

	// Физическая отправка через провайдера
	err = us.emailProvider.SendVerificationCode(email, code)
	if err != nil {
		return  err
	}

	return nil
}

// NewUserService создает экземпляр сервиса с необходимыми зависимостями
func NewUserService(userRepo repositories.UserRepositoryInter, emailProvider *utils.MailService, passwordHasher *utils.PasswordHasher) *userService {
	return &userService{userRepo: userRepo, emailProvider: emailProvider, passHasher: passwordHasher}
}