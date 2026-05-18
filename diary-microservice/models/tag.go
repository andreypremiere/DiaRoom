package models

import (
	"github.com/google/uuid"
)

type Tag struct {
	Id uuid.UUID `json:"id"`
	RoomId uuid.UUID `json:"roomId"`
	Name string `json:"name"`
	Color int64 `json:"color"`
}
