package requests

import "github.com/google/uuid"

type UpdatingMessage struct {
	MessageID uuid.UUID `json:"messageId"`
	Status    string `json:"status"`
}