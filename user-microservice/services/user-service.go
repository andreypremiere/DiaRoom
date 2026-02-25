package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	// "fmt"
	"user-microservice/models"
	"user-microservice/repositories"
	"user-microservice/utils"

	"github.com/google/uuid"
)

type UserServiceInter interface {
	AddUser(ctx context.Context, user *models.RegisterUser) (uuid.UUID, error)
}

type userService struct {
	userRepo repositories.UserRepositoryInter
	smsProvider utils.SmsProvider
}

func (us userService) AddUser(ctx context.Context, user *models.RegisterUser) (uuid.UUID, error) {
	newUUID := uuid.New()

	// Проверка полей на пустые значения (для user)
	if user.NumberPhone == "" {
		return uuid.Nil, errors.New("NumberPhone cannot be empty")
	}

	// Добавляем пользователя в базу данных через репозиторий
	err := us.userRepo.AddUser(ctx, newUUID, user.NumberPhone)
	if err != nil {
		fmt.Println("Ошибка добавления пользователя в базу данных", err)
		return uuid.Nil, err
	}

	// Создаем контекст с временем ответа
	reqCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()

	// Создаем необходимые параметры для запроса в room-microservice для создания комнаты
	url := "http://room-microservice:81/newRoom"
	roomCreating := models.RoomCreating{UserId: newUUID, RoomName: user.RoomName, RoomNameId: user.RoomId}
	// Сериализуем данные в json
	body, err := json.Marshal(roomCreating)
	if err != nil {
		return uuid.Nil, err
	}
	// Создаем запрос c контекстом в room-microservice для создания комнаты
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
        return uuid.Nil, err
    }
	req.Header.Set("Content-Type", "application/json")
	// Создаем клиента для отправки запроса
	client := &http.Client{}
	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка при обращении к микросервису rooms", err.Error())
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	//  Проверяем статус ответа
    if resp.StatusCode != http.StatusOK {
        fmt.Println("Внешний сервис вернул ошибку: ", resp.StatusCode)
		
		// Удаляем пользователя, если произошла ошибка
		err := us.DeleteUserById(ctx, newUUID)
		if err != nil {
			fmt.Println("Ошибка удаления пользователя при ошибке его создания. БАГ.", err.Error())
			return uuid.Nil, err
		} else {
			fmt.Println("Пользователь был удален")
		}
		return uuid.Nil, errors.New("Микросервис вернул плохой статус код")
    } 

	// Генерируем код
	code, err := utils.GenerateCode()
	if err != nil {
		return uuid.Nil, err
	}

	// Кладем код в базу данных Redis
	err = us.AddCodeWithTimeout(ctx, newUUID, code)
	if err != nil {
		return uuid.Nil, err
	}

	// Отправляем код
	err = us.smsProvider.SendCode(user.NumberPhone, code)
	if err != nil {
		return uuid.Nil, err
	}

	return newUUID, nil
}

func (us *userService) DeleteUserById(ctx context.Context, userId uuid.UUID) error {
	criticalCtx := context.WithoutCancel(ctx)
	err := us.userRepo.DeleteUserById(criticalCtx, userId)
	return err
}

func (us *userService) AddCodeWithTimeout(ctx context.Context, userId uuid.UUID, code string) error {
	err := us.userRepo.AddCodeWithTimeout(ctx, userId, code)
	return err
}

func NewUserService(userRepo repositories.UserRepositoryInter, smsProvider utils.SmsProvider) *userService {
	return &userService{userRepo: userRepo, smsProvider: smsProvider}
}