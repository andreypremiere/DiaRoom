package models

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"roomId"`

	ParentID *uuid.UUID `json:"parentId"`

	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}