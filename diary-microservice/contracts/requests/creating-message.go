package requests

import (
	"diary-microservice/models"

	"github.com/google/uuid"
)

type MessageCreateRequest struct {
	// Тип сообщения (standard, voice_note, video_note)
	MsgType string `json:"msgType"`

	// Текст сообщения
	Content *string `json:"content"`

	// Список метаданных вложений
	Attachments []AttachmentRequest `json:"attachments"`

	// Внешние ID (опционально)
	WorkshopFolderId  *uuid.UUID `json:"workshopFolderId"`
	PublicationPostId *uuid.UUID `json:"publicationPostId"`

	Tags []*models.Tag `json:"tags"`
}

// AttachmentRequest - метаданные вложения для генерации ссылок S3
type AttachmentRequest struct {
	// Тип (photo, video и т.д.)
	AttType string `json:"attType"`

	// Размер в байтах для валидации на сервере
	FileSizeBytes int64 `json:"fileSizeBytes"`

	// Длительность в миллисекундах (для аудио/видео)
	Duration *int64 `json:"duration"`

	// Mime-тип для правильных заголовков в S3
	MimeType string `json:"mimeType"`
}