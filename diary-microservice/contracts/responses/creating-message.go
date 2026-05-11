package responses

import "github.com/google/uuid"

// MessageCreateResponse - ответ сервера с подписанными ссылками
type MessageCreateResponse struct {
	MessageID   uuid.UUID            `json:"messageId"`
	Status      string               `json:"status"`
	UploadItems []*AttachmentUploadItem `json:"uploadItems"`
}

// AttachmentUploadItem - содержит ID вложения и ссылки для загрузки
type AttachmentUploadItem struct {
	AttachmentID        uuid.UUID `json:"attachmentId"`
	PresignedURL        string    `json:"presignedUrl"`
	PresignedPreviewURL *string   `json:"presignedPreviewUrl"` 
}