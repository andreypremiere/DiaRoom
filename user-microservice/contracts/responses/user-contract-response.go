package responses

import "github.com/google/uuid"

type UserVerifyResponse struct {
	UserId uuid.UUID `json:"userId"`
	AccessToken string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}