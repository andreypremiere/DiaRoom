package requests

import "github.com/google/uuid"

type UserCreatingContract struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Структура для верификации пользователя
type VerifyUserById struct {
	UserId uuid.UUID `json:"userId"`
	Code   string    `json:"code"`
}

type UserLogin struct {
	Email string `json:"email"`
	Password string `json:"password"`
}