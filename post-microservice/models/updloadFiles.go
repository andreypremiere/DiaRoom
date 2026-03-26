package models

import "github.com/google/uuid"

// GenerateUrlsRequest — модель входящего запроса от Flutter
type GenerateUrlsRequest struct {
	PostID uuid.UUID    `json:"postId"`
	Files  []UploadFile `json:"files"`
}

type UploadFile struct {
	UploadID    uuid.UUID `json:"uploadId"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

// GenerateUrlsResponse — модель ответа сервера
type GenerateUrlsResponse struct {
	Files []GeneratedURL `json:"files"`
}

type GeneratedURL struct {
	UploadID     uuid.UUID `json:"uploadId"`
	PublicURL    string `json:"publicUrl"`
	PresignedURL string `json:"presignedUrl"`
}