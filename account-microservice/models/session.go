package models

import (
	"time"

	"github.com/google/uuid"
)

type BaseSession struct {
	UserId uuid.UUID
	RefreshToken string
	UserAgent string
	ExpiresAt time.Time
}

type SessionWithRoomId struct {
    BaseSession
    RoomId uuid.UUID
}