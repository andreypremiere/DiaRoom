package responses

import (
	"workshop-microservice/models"

	"github.com/google/uuid"
)

type FolderShow struct {
	ID     uuid.UUID `json:"id"`
	RoomID uuid.UUID `json:"roomId"`

	ParentID *uuid.UUID `json:"parentId"`

	Name string `json:"name"`
}

func (f *FolderShow) FromModel(folder *models.Folder) *FolderShow {
	f.ID = folder.ID
	f.RoomID = folder.RoomID
	f.ParentID = folder.ParentID
	f.Name = folder.Name
	return f
}