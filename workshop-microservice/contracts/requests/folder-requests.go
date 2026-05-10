package requests

import (
	"workshop-microservice/models"

	"github.com/google/uuid"
)

type CreateFolder struct {
	RoomId *uuid.UUID 
	FolderName string `json:"folderName"`
	ParentId *uuid.UUID `json:"parentId"`
}

func (req CreateFolder) ToDomain() *models.Folder {
    return &models.Folder{
        RoomID:   *req.RoomId, 
        ParentID: req.ParentId,
        Name:     req.FolderName,
    }
}

type MoveFolder struct {
	TargetId uuid.UUID `json:"targetId"`
	DestinationId *uuid.UUID `json:"destinationId"`
}