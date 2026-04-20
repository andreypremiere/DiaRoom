package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"post-microservice/contracts/responses"
	"time"

	"github.com/google/uuid"
)

type AccountClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAccountClient(baseURL string) *AccountClient {
	return &AccountClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 500 * time.Millisecond, // Обязательно ставим таймаут!
		},
	}
}

func (c *AccountClient) GetAuthor(ctx context.Context, roomId uuid.UUID) (*responses.RoomInfo, error) {
	url := fmt.Sprintf("%s/getRoomInfoInternal", c.baseURL)

	body, _ := json.Marshal(map[string]uuid.UUID{"room_id": roomId})
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("account service error: status %d", resp.StatusCode)
	}

	var result responses.RoomInfo
	json.NewDecoder(resp.Body).Decode(&result)

	return &result, nil
}

// Конкретный метод для получения данных авторов
func (c *AccountClient) GetAuthorsBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]responses.RoomInfo, error) {
	url := fmt.Sprintf("%s/getRoomsInfoInternal", c.baseURL)

	// Здесь на самом деле room_ids, опечатка, поиск идет по room_id в дальнейшей цепочке
	body, _ := json.Marshal(map[string][]uuid.UUID{"user_ids": ids})
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("account service error: status %d", resp.StatusCode)
	}

	var result map[uuid.UUID]responses.RoomInfo
	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}