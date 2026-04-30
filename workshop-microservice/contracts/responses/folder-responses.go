package responses

import "github.com/google/uuid"

type FolderShow struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"roomId"`

	ParentID *uuid.UUID `json:"parentId"`

	Name string `json:"name"`
}