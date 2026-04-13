package responses

import "github.com/google/uuid"

type RoomInfo struct {
	AvatarUrl string `json:"avatarUrl"`
	RoomName string `json:"roomName"`
}

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