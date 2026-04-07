package services

import (
	"account-microservice/contracts/account/requests"
	"account-microservice/repositories"
	"account-microservice/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypremiere/jwtmanager"
	"github.com/google/uuid"
)

type AccountService struct {
	accountRepo   repositories.AccountRepository
	emailProvider *utils.MailService
	passHasher    *utils.PasswordHasher
	jwtManager    *jwtmanager.JWTManager
}

func (as *AccountService) NewAccount(ctx context.Context, newUser *requests.CreatingAccount) (*uuid.UUID, error) {
	newUserId := uuid.New()
	newRoomId := uuid.New()

	if newUser.Email == "" {
		return nil, errors.New("Поле Email оказалось пустым")
	}
	if newUser.Password == "" {
		return nil, errors.New("Поле Password оказалось пустым")
	}

	// Добавить проверку требования пароля

	hashPassword, err := as.passHasher.HashPassword(newUser.Password)
	if err != nil {
		return nil, errors.New("Ошибка во время создания хеша пароля")
	}

	roomUniqueId := fmt.Sprintf("USER-%s", newRoomId)
	roomName := roomUniqueId

	err = as.accountRepo.NewAccount(ctx, newUser.Email, newUserId, newRoomId, roomUniqueId, roomName, hashPassword)
	if err != nil {
		return nil, err
	}
	
	as.GenerateAndSendCode(newUserId, newUser.Email)

	return &newUserId, nil
}

func (as *AccountService) GenerateAndSendCode(userId uuid.UUID, email string) {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        err := as.doGenerateAndSend(ctx, userId, email)
        if err != nil {
            fmt.Printf("[ASYNC ERROR] Ошибка для юзера %s: %v\n", userId, err)
        }
    }()
}

func (as *AccountService) doGenerateAndSend(ctx context.Context, userId uuid.UUID, email string) error {
    code, err := utils.GenerateCode()
    if err != nil {
        return fmt.Errorf("генерация кода: %w", err)
    }

    if err := as.accountRepo.AddCodeWithTimeout(ctx, userId, code); err != nil {
        return fmt.Errorf("запись в Redis: %w", err)
    }

    if err := as.emailProvider.SendVerificationCode(email, code); err != nil {
        return fmt.Errorf("отправка почты: %w", err)
    }

    return nil
}

func NewAccountService(
	accountRepo *repositories.AccountRepository,
	emailProvider *utils.MailService,
	passwordHasher *utils.PasswordHasher,
	jwtManager *jwtmanager.JWTManager,
) *AccountService {
	return &AccountService{
		accountRepo: *accountRepo,
		emailProvider: emailProvider,
		passHasher:    passwordHasher,
		jwtManager:    jwtManager,
	}
}