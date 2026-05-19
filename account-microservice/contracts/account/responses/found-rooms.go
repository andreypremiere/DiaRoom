package responses

import "github.com/google/uuid"

type FoundRooms struct {
	Rooms []*RoomInfoExpanded `json:"rooms"`
}

type RoomInfoExpanded struct {
	Id uuid.UUID `json:"id"`
	RoomUniqueId string `json:"roomUniqueId"`
	Nickname string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}