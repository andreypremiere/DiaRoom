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

func (s *WorkshopService) GetFolder(ctx context.Context, folderID uuid.UUID) (*responses.Root, error) {
    folders, err := s.repo.GetFolder(ctx, folderID)
    if err != nil {
        return nil, err
    }

    result := &responses.Root{
        Folders: make([]*responses.FolderShow, 0, len(folders)),
    }

    for _, f := range folders {
        show := &responses.FolderShow{}
        result.Folders = append(result.Folders, show.FromModel(f))
    }

    return result, nil
}

func (s *WorkshopService) MoveFolder(ctx context.Context, roomID uuid.UUID, moving *requests.MoveFolder) error {
    // Делегируем проверку и обновление репозиторию
    return s.repo.MoveFolder(ctx, roomID, moving.TargetId, moving.DestinationId)
}

func (s *WorkshopService) RenameFolder(ctx context.Context, roomID, folderID uuid.UUID, newName string) error {
	if newName == "" {
		return apperrors.ErrInvalidInput
	}

	return s.repo.RenameFolder(ctx, folderID, roomID, newName)
}

func (s *WorkshopService) GetRoot(ctx context.Context, roomId uuid.UUID) (*responses.Root, error) {
	folders, err := s.repo.GetRootFolders(ctx, roomId)
	if err != nil {
		return nil, err
	}

	resultFolders := make([]*responses.FolderShow, 0)
	if len(folders) != 0 {
		for _, item := range folders {
			f := &responses.FolderShow{}
			resultFolders = append(resultFolders, f.FromModel(item))
		}
	}

	return &responses.Root{Folders: resultFolders}, nil
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