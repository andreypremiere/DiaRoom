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

func GetRoomIdByUserId(userId uuid.UUID) (uuid.UUID, error) {
	client := &http.Client{
		Timeout: time.Second*2,
	}

	body, err := json.Marshal(map[string]uuid.UUID{"userId": userId})
	if err != nil {
		return uuid.Nil, errors.New("Ошибка преобразования id в json")
	}

	req, err := http.NewRequest(
		http.MethodPost, 
		configurations.GetRoomIdByUserId, 
		bytes.NewBuffer(body))

	if err != nil {
		return uuid.Nil, errors.New("Ошибка создания запроса") 
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return uuid.Nil, errors.New("Ошибка выполнения запроса по получению id комнаты") 
	}
	
	defer resp.Body.Close()

	var roomId struct {RoomId uuid.UUID `json:"roomId"`} 
	var error struct {Error string `json:"error"`} 
	if resp.StatusCode == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&roomId)
		if err != nil {
			return uuid.Nil, errors.New("Ошибка чтения ответа") 
		}
		return roomId.RoomId, nil
	} else {
		err := json.NewDecoder(resp.Body).Decode(&error)
		if err != nil {
			return uuid.Nil, errors.New("Ошибка чтения ответа") 
		}
		return uuid.Nil, errors.New("Плохой ответ от room-microservice" + error.Error)
	}
}