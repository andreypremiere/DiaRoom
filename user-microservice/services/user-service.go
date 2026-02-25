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

	"github.com/google/uuid"
)

type UserServiceInter interface {
	AddUser(ctx context.Context, user *models.RegisterUser) (uuid.UUID, error)
}

type userService struct {
	userRepo repositories.UserRepositoryInter
}

func (us userService) AddUser(ctx context.Context, user *models.RegisterUser) (uuid.UUID, error) {
	newUUID := uuid.New()

	// Проверка полей на пустые значения
	if user.NumberPhone == "" {
		return newUUID, errors.New("NumberPhone cannot be empty")
	}

	// Добавляем пользователя в базу данных через репозиторий
	err := us.userRepo.AddUser(ctx, newUUID, user.NumberPhone)
	if err != nil {
		fmt.Println("Ошибка добавления пользователя в базу данных", err)
		return newUUID, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()

	url := "http://room-microservice:81/newRoom"
	roomCreating := models.RoomCreating{UserId: newUUID, RoomName: user.RoomName, RoomNameId: user.RoomId}
	body, err := json.Marshal(roomCreating)
	if err != nil {
		return newUUID, err
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
        return newUUID, err
    }
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка при обращении к микросервису rooms", err.Error())
		return newUUID, err
	}
	defer resp.Body.Close()

	// 5. Проверяем статус ответа
    if resp.StatusCode != http.StatusOK {
        fmt.Println("Внешний сервис вернул ошибку: ", resp.StatusCode)
		return newUUID, errors.New("Микросервис вернул плохой статус код")
    }

	return newUUID, nil

	// Если все успешно, вызываем генерацию кода

	// Отправляем код

	// Возвращаем id
}

func NewUserService(userRepo repositories.UserRepositoryInter) *userService {
	return &userService{userRepo: userRepo}
}