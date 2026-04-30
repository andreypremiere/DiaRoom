package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"roomId"`

	FolderID *uuid.UUID `json:"folderId"`

	Title      string `json:"title"`
	PreviewURL string `json:"previewUrl"`
	SizeBytes  int64  `json:"sizeBytes"`
	ItemType   string `json:"itemType"` // photo, video, canvas
	Status     string `json:"status"`      // uploading, ready, failed

	// хранит как байты,
	// позже парсится в VideoPayload/PhotoPayload/CanvasPayload позже.
	Payload json.RawMessage `json:"payload"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}