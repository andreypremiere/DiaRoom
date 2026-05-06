package responses

import (
	"encoding/json"
	"workshop-microservice/models"

	"github.com/google/uuid"
)

type ItemShow struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"roomId"`

	FolderID *uuid.UUID `json:"folderId"`

	Title      string `json:"title"`
	PreviewURL string `json:"previewUrl"`
	// SizeBytes  int64  `json:"sizeBytes"`
	ItemType   string `json:"itemType"` // photo, video, canvas
	Status     string `json:"status"`   // uploading, ready, failed
	// MimeType   string `json:"mimetype"`

	Payload json.RawMessage `json:"payload"`
}

func (i *ItemShow) FromModel(item *models.Item) *ItemShow {
	i.ID = item.ItemData.ID
	i.RoomID = item.ItemData.RoomID
	i.FolderID = item.ItemData.FolderID
	i.Title = item.ItemData.Title
	i.PreviewURL = item.ItemData.PreviewURL
	i.ItemType = item.ItemData.ItemType
	i.Status = item.ItemData.Status
	i.Payload = item.Payload
	return i
}