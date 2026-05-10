package requests

import "github.com/google/uuid"

type CreatingItemVideo struct {
	Title     string `json:"title"`
	MimeType  string `json:"mimeType"`
	FolderID  *uuid.UUID `json:"folderId"`
	SizeBytes int64  `json:"sizeBytes"`
	ItemType  string `json:"itemType"`
	Duration int64 `json:"duration"`
}