package responses

import (
	"encoding/json"

	"github.com/google/uuid"
)

type PostInfo struct {
	Id uuid.UUID `json:"postId"`
	RoomId uuid.UUID `json:"roomId"`
	PreviewUrl string `json:"previewUrl"`
	Category string `json:"categorySlug"`
	CanvasId uuid.UUID `json:"canvasId"`
	Title string `json:"title"`
	ViewsCount int `json:"viewsCount"`
	LikesCount int `json:"likesCount"`
}

type Post struct {
	RoomInfo
	PostInfo
}

type RoomInfo struct {
	AvatarUrl string `json:"avatarUrl"`
	RoomName string `json:"roomName"`
}

type ShowingPost struct {
	RoomInfo
	RoomId uuid.UUID `json:"roomId"`
	Canvas json.RawMessage `json:"payload"`
	Category string `json:"categorySlug"`
	Hashtags []string `json:"hashtags"`
	ViewsCount int `json:"viewsCount"`
	LikesCount int `json:"likesCount"`
}

type PostInfoPersonal struct {
	PostInfo
	Status string `json:"status"`
	StatusAi string `json:"statusAi"`
}