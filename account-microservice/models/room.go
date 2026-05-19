package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         uuid.UUID       `json:"userId" db:"user_id"`
	RoomName       string          `json:"roomName" db:"room_name"`
	RoomUniqueID   string          `json:"roomUniqueId" db:"room_unique_id"`
	AvatarURL      string          `json:"avatarUrl" db:"avatar_url"`
	BackgroundURL  string          `json:"backgroundUrl" db:"background_url"`
	Bio            string          `json:"bio" db:"bio"`
	Settings       json.RawMessage `json:"settings" db:"settings"` 
	FollowersCount int             `json:"followersCount" db:"followers_count"`
	FollowingCount int             `json:"followingCount" db:"following_count"`
	CreatedAt      time.Time       `json:"createdAt" db:"created_at"`
}