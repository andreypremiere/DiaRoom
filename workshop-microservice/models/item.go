package models

import (
	"encoding/json"
	"fmt"
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
	MimeType   string `json:"mimetype"`

	Payload json.RawMessage `json:"payload"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ImagePayload struct {
	PublicURL string `json:"publicUrl"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type VideoPayload struct {
	PublicURL string `json:"publicUrl"`
	Duration  int    `json:"duration"`
}

func ParseItemPayload(item Item) (interface{}, error) {
	switch item.ItemType {
	case "photo":
		var p ImagePayload
		if err := json.Unmarshal(item.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	case "video":
		var p VideoPayload
		if err := json.Unmarshal(item.Payload, &p); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("неизвестный тип: %s", item.ItemType)
	}
}