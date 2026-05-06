package responses

import "github.com/google/uuid"

type CreatingItemVideo struct {
	ItemId               uuid.UUID `json:"itemId"`
	PresignedUrlPreview  string `json:"presignedUrlPreview"`
	PresignedUrlOriginal string `json:"presignedUrlOriginal"`
}