package responses

import "github.com/google/uuid"

type LoginResponse struct {
	UserId uuid.UUID `json:"userId"`
	Email string `json:"email"`
}