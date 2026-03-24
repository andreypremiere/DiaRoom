package models

import (
	"github.com/google/uuid"
)


type CreatePostRequest struct {
    Post    PostData    `json:"post"`
    Preview PreviewData `json:"preview"`
}

type PostData struct {
    RoomID       uuid.UUID `json:"roomId"` // Если передаешь из Flutter
    Title        string    `json:"title"`
    PostStatus   string    `json:"postStatus"`
    AiStatus     string    `json:"aiStatus"`
    CategorySlug string    `json:"categorySlug"`
    Hashtags     []string  `json:"hashtags"`
}

type PreviewData struct {
    UploadID    string `json:"uploadId"`
    Filename    string `json:"filename"`
    ContentType string `json:"contentType"`
    Size        int64  `json:"size"`
}

type CreatePostResponse struct {
    PostID  uuid.UUID           `json:"postId"`
    Preview PreviewLinksResponse `json:"preview"`
}

type PreviewLinksResponse struct {
    PublicURL    string `json:"publicUrl"`
    PresignedURL string `json:"presignedUrl"`
}

type CreatePostInternal struct {
    RoomID     uuid.UUID
    CategoryID int
    Title      string
    Status     string
    AiStatus   string
}