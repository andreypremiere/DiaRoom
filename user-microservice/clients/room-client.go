package clients

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/google/uuid"
)

type RoomClient struct {
    host       string
    httpClient *http.Client
}

func NewRoomClient(host string) *RoomClient {
    return &RoomClient{
        host: host,
        httpClient: &http.Client{
            Timeout: 3 * time.Second,
        },
    }
}

// GetRoomId запрашивает ID комнаты у микросервиса Room
func (c *RoomClient) GetRoomId(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
    url := fmt.Sprintf("%s/getRoomIdByUserId", c.host)

    // Подготовка тела запроса
    requestBody, err := json.Marshal(map[string]uuid.UUID{
        "userId": userID,
    })
    if err != nil {
        return uuid.Nil, fmt.Errorf("ошибка маршалинга: %w", err)
    }

    // Создаем запрос с контекстом (важно для отмены запроса)
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
    if err != nil {
        return uuid.Nil, fmt.Errorf("ошибка создания запроса: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    // Выполняем
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return uuid.Nil, fmt.Errorf("ошибка при выполнении запроса к Room-Service: %w", err)
    }
    defer resp.Body.Close()

    // Проверяем статус
    if resp.StatusCode != http.StatusOK {
        return uuid.Nil, fmt.Errorf("room-service вернул ошибку: статус %d", resp.StatusCode)
    }

    // Декодируем ответ
    var response struct {
        RoomId uuid.UUID `json:"roomId"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return uuid.Nil, fmt.Errorf("ошибка декодирования ответа: %w", err)
    }

    return response.RoomId, nil
}