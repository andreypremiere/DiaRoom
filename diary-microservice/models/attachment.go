package models

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID            uuid.UUID `json:"id" db:"id"`
	MessageID     uuid.UUID `json:"messageId" db:"message_id"`
	AttType       string    `json:"attType" db:"att_type"` // photo, video, voice_note, video_note
	S3Key         string    `json:"s3Key" db:"s3_key"`
	PreviewS3Key  *string   `json:"previewS3Key,omitempty" db:"preview_s3_key"` // Новое поле
	FileSizeBytes int64     `json:"fileSizeBytes" db:"file_size_bytes"`
	Duration      *int64    `json:"duration,omitempty" db:"duration"` // BIGINT -> int64
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}
