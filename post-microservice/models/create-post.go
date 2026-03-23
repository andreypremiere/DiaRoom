package models

import (
	"encoding/json"

	"github.com/google/uuid"
)


type CreatePostRequest struct {
    RoomID     uuid.UUID `json:"roomId"`
    CategorySlug string    `json:"categorySlug"`
    Title      string    `json:"title"`
}

type CreatePostResponse struct {
    PostID uuid.UUID `json:"postId"`
    Status string    `json:"status"`
}

type PublishPostRequest struct {
	PostID     uuid.UUID       `json:"postId"`
	Payload    json.RawMessage `json:"payload"` // Сырой JSON, идеально для JSONB
	PreviewURL *string         `json:"previewUrl,omitempty"`
	Hashtags   []string        `json:"hashtags,omitempty"`
}

type PublishPostResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}