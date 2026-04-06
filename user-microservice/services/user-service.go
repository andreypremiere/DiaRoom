package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"user-microservice/clients"
	"user-microservice/contracts/requests"
	"user-microservice/contracts/responses"
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
	VerifyCode(ctx context.Context, verifyUser requests.VerifyUserById) (*responses.UserVerifyResponse, error)
	// LoginUser инициирует процесс входа через отправку кода
	LoginUser(ctx context.Context, user requests.UserLogin) (*responses.UserVerifyResponse, *models.BaseUser,  error)
	RepeatSendingCode(ctx context.Context, userId uuid.UUID) error
	RefreshSession(ctx context.Context, oldRefreshToken string) (*responses.UserVerifyResponse, error)
	Logout(ctx context.Context, refreshToken string) error

}

// userService реализует бизнес-логику с использованием репозитория и SMS-провайдера
type userService struct {
	userRepo    repositories.UserRepositoryInter
	emailProvider *utils.MailService
	passHasher *utils.PasswordHasher
	jwtManager *jwtmanager.JWTManager
	roomClient *clients.RoomClient
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

func (us *userService) RepeatSendingCode(ctx context.Context, userId uuid.UUID) error {
	// Найти пользователя по userId, получить email
	email, err := us.userRepo.GetEmailByUserId(ctx, userId)
	if err != nil {
		return errors.New("Ошибка: пользователь не найден или не получилось найти")
	}

	err2 := us.GenerateAndSendCode(ctx, userId, email)
	if err2 != nil {
		return err
	}

	return nil
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

func (us *userService) RefreshSession(ctx context.Context, oldRefreshToken string) (*responses.UserVerifyResponse, error) {
    // 1. Проверяем токен в БД и получаем userId
    session, err := us.userRepo.GetSessionByToken(ctx, oldRefreshToken)
    if err != nil {
        return nil, fmt.Errorf("сессия не найдена: %w", err)
    }

	

    // 2. Проверяем срок годности
    if session.ExpiresAt.Before(time.Now()) {
        _ = us.userRepo.DeleteRefreshToken(ctx, oldRefreshToken) // Чистим мусор
        return nil, fmt.Errorf("срок действия refresh токена истек")
    }

    // 3. Генерируем НОВУЮ пару
    newAccessToken, _ := us.jwtManager.Generate(session.UserId.String())
    newRefreshToken := uuid.New().String()
    newExpiresAt := time.Now().Add(30 * 24 * time.Hour) // Продлеваем на месяц

    // 4. Ротация в базе: заменяем старый на новый
    err = us.userRepo.UpdateRefreshToken(ctx, oldRefreshToken, newRefreshToken, newExpiresAt)
    if err != nil {
        return nil, fmt.Errorf("не удалось обновить сессию: %w", err)
    }

    return &responses.UserVerifyResponse{
        UserId:       session.UserId,
        AccessToken:  newAccessToken,
        RefreshToken: newRefreshToken,
    }, nil
}

func (us *userService) Logout(ctx context.Context, refreshToken string) error {
    return us.userRepo.DeleteRefreshToken(ctx, refreshToken)
}

// VerifyCode проверяет код и генерирует JWT при успехе
func (us *userService) VerifyCode(ctx context.Context, verifyUser requests.VerifyUserById) (*responses.UserVerifyResponse, error) {
	// Получение эталонного кода из Redis
	if verifyUser.Code == "" || len(verifyUser.Code) != 6 {
		return nil, errors.New("Неверный формат кода")
	}

	gotCode, err := us.userRepo.GetValueByKey(ctx, verifyUser.UserId)
	if err != nil {
		return nil, errors.Join(errors.New("Ошибка получения кода из базы данных redis: "), err)
	}

	// Сравнение кодов
	if gotCode != verifyUser.Code {
		return nil, errors.New("Введенный код не совпадает с полученным")
	}

    err = us.userRepo.SetUserVerified(ctx, verifyUser.UserId)
    if err != nil {
        return nil, fmt.Errorf("не удалось обновить статус верификации: %w", err)
    }

	// Логика внутри a.userService.VerifyCode:
	accessToken, _ := us.jwtManager.Generate(verifyUser.UserId.String()) // Твой JWT на 15 минут
	refreshToken := uuid.New().String() // Генерируем уникальную строку
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Живет 30 дней

	// Сохранить в БД (INSERT INTO sessions...)
	err = us.userRepo.AddRefreshToken(ctx, verifyUser.UserId, refreshToken, "some info user agent", expiresAt)
	if err != nil {
		return  nil, errors.New("Ошибка добавления токена обновления в бд")
	}

	response := &responses.UserVerifyResponse{UserId: verifyUser.UserId, AccessToken: accessToken, RefreshToken: refreshToken}
	
	return response, nil
}

// LoginUser ищет пользователя и отправляет ему новый код для входа
func (us *userService) LoginUser(ctx context.Context, user requests.UserLogin) (*responses.UserVerifyResponse, *models.BaseUser,  error) {
	if user.Email == "" || user.Password == "" {
		return nil, nil, errors.New("Поля не должны быть пустыми")
	}

	userCkecked, err := us.userRepo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, nil, err
	}

	if !userCkecked.IsActivated {
		return nil, &models.BaseUser{Id: userCkecked.Id, Email: userCkecked.Email, IsActivated: userCkecked.IsActivated}, nil
	}

	if (!us.passHasher.ComparePassword(user.Password, userCkecked.HashPassword)) {
		return nil, nil, errors.New("Пароли не совпадают")
	}

	// Логика внутри a.userService.VerifyCode:
	accessToken, _ := us.jwtManager.Generate(userCkecked.Id.String()) // Твой JWT на 15 минут
	refreshToken := uuid.New().String() // Генерируем уникальную строку
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Живет 30 дней

	// Сохранить в БД (INSERT INTO sessions...)
	err = us.userRepo.AddRefreshToken(ctx, userCkecked.Id, refreshToken, "some info user agent", expiresAt)
	if err != nil {
		return  nil, nil, errors.New("Ошибка добавления токена обновления в бд")
	}

	response := &responses.UserVerifyResponse{UserId: userCkecked.Id, AccessToken: accessToken, RefreshToken: refreshToken}
	
	return response, nil, nil 
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
func NewUserService(userRepo repositories.UserRepositoryInter, emailProvider *utils.MailService, passwordHasher *utils.PasswordHasher,
	jwtManager *jwtmanager.JWTManager, roomClient *clients.RoomClient) *userService {
	return &userService{userRepo: userRepo, emailProvider: emailProvider, passHasher: passwordHasher,
	jwtManager: jwtManager, roomClient: roomClient,}
}