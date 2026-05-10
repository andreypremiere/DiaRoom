package responses

import "github.com/google/uuid"

type CreatingItemPhoto struct {
	ItemId               uuid.UUID `json:"itemId"`
	PresignedUrlPreview  string `json:"presignedUrlPreview"`
	PresignedUrlOriginal string `json:"presignedUrlOriginal"`
}