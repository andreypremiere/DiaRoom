package models

import "github.com/google/uuid"

type BaseUser struct {
	Id uuid.UUID `json:"id"`
	NumberPhone string `json:"numberPhone"`
}

type RegisterUser struct {
	BaseUser
	RoomId   string `json:"roomId"`
	RoomName string `json:"roomName"`
}

type RoomCreating struct {
	UserId uuid.UUID `json:"userId"`
	RoomNameId string `json:"roomNameId"`
	RoomName string `json:"roomName"`
}

type VerifyUserById struct {
	UserId uuid.UUID `json:"userId"`
	Code string `json:"code"`
}

// type DataForRegister struct {
// 	// Id string `json:"id"`
// 	Code int `json:"code"`
// 	HashEmailCode string `json:"hashEmailCode"`
// }