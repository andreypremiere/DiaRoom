package services

import (
	"context"
	apperrors "workshop-microservice/app-errors"
	"workshop-microservice/contracts/requests"
	"workshop-microservice/contracts/responses"
	"workshop-microservice/repositories"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type WorkshopService struct {
	repo *repositories.WorkshopRepository
	s3Client *s3.Client
}

func (s *WorkshopService) CreateFolder(ctx context.Context, newFolder *requests.CreateFolder) (*responses.FolderShow, error) {
	if newFolder.RoomId == nil || *newFolder.RoomId == uuid.Nil {
		return nil, apperrors.ErrInvalidInput
	}

	if newFolder.FolderName == "" {
		return nil, apperrors.ErrInvalidInput
	}

	folder, err := s.repo.CreateFolder(ctx, newFolder.ToDomain())
	if err != nil {
		return nil, err
	}

	return &responses.FolderShow{
		ID: folder.ID, 
		RoomID: folder.RoomID, 
		ParentID: folder.ParentID, 
		Name: folder.Name,
		}, nil
} 

func NewWorkshopService(repo *repositories.WorkshopRepository, s3 *s3.Client) *WorkshopService {
	return &WorkshopService{repo: repo, s3Client: s3}
}