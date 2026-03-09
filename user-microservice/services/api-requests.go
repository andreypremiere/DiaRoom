package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"user-microservice/configurations"

	"github.com/google/uuid"
)

// GetRoomIdByUserId выполняет внешний HTTP-запрос к room-microservice для получения ID комнаты
func GetRoomIdByUserId(userId uuid.UUID) (uuid.UUID, error) {
	// Настройка клиента с таймаутом в 2 секунды
	client := &http.Client{
		Timeout: time.Second * 2,
	}

	// Подготовка тела запроса в формате JSON
	body, err := json.Marshal(map[string]uuid.UUID{"userId": userId})
	if err != nil {
		return uuid.Nil, errors.New("Ошибка преобразования id в json")
	}

	// Создание POST запроса к другому микросервису
	req, err := http.NewRequest(
		http.MethodPost,
		configurations.GetRoomIdByUserId,
		bytes.NewBuffer(body))

	if err != nil {
		return uuid.Nil, errors.New("Ошибка создания запроса")
	}

	// Установка обязательного заголовка типа контента
	req.Header.Set("Content-Type", "application/json")

	// Выполнение сетевого запроса
	resp, err := client.Do(req)
	if err != nil {
		return uuid.Nil, errors.New("Ошибка выполнения запроса по получению id комнаты")
	}

	// Закрытие тела ответа после завершения работы функции
	defer resp.Body.Close()

	var roomId struct {
		RoomId uuid.UUID `json:"roomId"`
	}
	var errorResp struct {
		Error string `json:"error"`
	}

	// Обработка успешного ответа от сервиса
	if resp.StatusCode == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&roomId)
		if err != nil {
			return uuid.Nil, errors.New("Ошибка чтения ответа")
		}
		return roomId.RoomId, nil
	}

	// Обработка ошибок, полученных от внешнего сервиса
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	if err != nil {
		return uuid.Nil, errors.New("Ошибка чтения ответа")
	}
	return uuid.Nil, errors.New("Плохой ответ от room-microservice: " + errorResp.Error)
}