package responses

import "github.com/google/uuid"

type RoomResponse struct {
	RoomUniqueId   string   `json:"roomUniqueId"`
	RoomName       string   `json:"roomName"`
	ListCategory   []string `json:"listCategory"`
	Bio            string   `json:"bio"`
	AvatarPath     string   `json:"avatarPath"`
	BackgroundPath string   `json:"backgroundPath"`
	CountFollowers int      `json:"countFollowers"`
	CountFollowing int      `json:"countFollowing"`
}

type UpdateRoomResponse struct {
	PresignedUrlAvatar     string `json:"presignedUrlAvatar"`
	PresignedUrlBackground string `json:"presignedUrlBackground"`
}

type RoomInfo struct {
	Id  uuid.UUID `json:"roomId"`
	AvatarUrl string `json:"avatarUrl"`
	RoomName  string `json:"roomName"`
}
