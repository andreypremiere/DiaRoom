package models

import "github.com/google/uuid"

type BaseUser struct {
	Id uuid.UUID `json:"id"`
	NumberPhone string `json:"numberPhone"`
	RoomId   string `json:"roomId"`
}

// Структура для регистрации пользователя
type RegisterUser struct {
	BaseUser
	RoomName string `json:"roomName"`
}

// Структура для создания комнаты
type RoomCreating struct {
	UserId uuid.UUID `json:"userId"`
	RoomNameId string `json:"roomNameId"`
	RoomName string `json:"roomName"`
}

// Структура для верификации пользователя
type VerifyUserById struct {
	UserId uuid.UUID `json:"userId"`
	Code string `json:"code"`
}
