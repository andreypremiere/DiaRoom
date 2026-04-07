package requests

import "github.com/google/uuid"

type VerifyUser struct {
	UserId uuid.UUID `json:"userId"`
	Code   string    `json:"code"`
	DeviceInfo string `json:"deviceInfo"`
}

type LoginUser struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

