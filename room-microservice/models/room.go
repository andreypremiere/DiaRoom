package models

import "github.com/google/uuid"

type BaseRoom struct {
	Id uuid.UUID `json:"id"`
	UserId     uuid.UUID `json:"userId"`
	RoomName   string `json:"roomName"`
	RoomNameId string `json:"roomNameId"`
}