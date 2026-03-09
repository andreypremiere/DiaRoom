package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type BaseRoom struct {
	Id uuid.UUID `json:"id"`
	UserId     uuid.UUID `json:"userId"`
	RoomName   string `json:"roomName"`
	RoomNameId string `json:"roomNameId"`
}

// Используется для загрузки страницы комнаты и получения полных данных о комнате
type RoomExpanded struct {
	BaseRoom
	Categories []Category `json:"categories"`
	AvatarUrl *string `json:"avatarUrl"`
	Bio *string `json:"bio"`
	Settings json.RawMessage `json:"settings"`
	FollowersCount int `json:"followersCount"`
	FollowingCount int `json:"followingCount"`
}

type Category struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}