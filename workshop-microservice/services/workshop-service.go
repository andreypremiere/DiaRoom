package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	apperrors "workshop-microservice/app-errors"
	"workshop-microservice/contracts/requests"
	"workshop-microservice/contracts/responses"
	"workshop-microservice/models"
	"workshop-microservice/repositories"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type WorkshopService struct {
	repo *repositories.WorkshopRepository
	s3Client *s3.Client
	s3PresignedClient *s3.PresignClient
	bucketName string
	endpoint string
}



func (s *WorkshopService) ValidateFolderAccess(ctx context.Context, folderID uuid.UUID, roomID uuid.UUID) (bool, error) {
    inRoom, err := s.repo.IsFolderInRoom(ctx, folderID, roomID)
    if err != nil {
        return inRoom, err
    }

    if !inRoom {
        return false, nil
    }

    return true, nil
}

func (s *WorkshopService) GenerateUploadUrls(ctx context.Context, key string, mimeType string) (string, string, error) {
	// Генерируем публичную ссылку
	// Формат для Yandex: https://storage.yandexcloud.net/bucket/key
	publicUrl := fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucketName, key)

	// Генерируем Presigned URL для PUT-запроса
	// Мы указываем ContentType, чтобы S3 проверял его при загрузке
	presignedRequest, err := s.s3PresignedClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(mimeType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(15 * time.Minute) 
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to sign request: %w", err)
	}

	return publicUrl, presignedRequest.URL, nil
}

func (s *WorkshopService) getExtensionFromMime(mimeType string) string {
    parts := strings.Split(mimeType, "/")
    if len(parts) < 2 {
        return "bin" 
    }
    
    return parts[1]
}

func (s *WorkshopService) getRelativePath(fullUrl string) string {
    // Формируем базу, которую нужно удалить "https://storage.yandexcloud.net/"
    base := fmt.Sprintf("%s/", s.endpoint)
    
    return strings.TrimPrefix(fullUrl, base)
}

func (s *WorkshopService) CreateImageItem(ctx context.Context, roomId uuid.UUID, item *requests.CreatingItemPhoto) (*responses.CreatingItemPhoto, error) {
	if item.FolderID != nil {
		inRoom, err := s.ValidateFolderAccess(ctx, *item.FolderID, roomId)
		if err != nil {
			return nil, err
		}
		if !inRoom {
			return nil, apperrors.ErrNotFound
		}
	}

	ext := s.getExtensionFromMime(item.MimeType)

	itemId := uuid.New()

	keyPreview := fmt.Sprintf("%s/%s/%s.%s", roomId, itemId, uuid.New(), ext)
	keyOriginal := fmt.Sprintf("%s/%s/%s.%s", roomId, itemId, uuid.New(), ext)

	publicUrlPreview, presignedUrlPreview, err := s.GenerateUploadUrls(ctx, keyPreview, item.MimeType)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	publicUrlOriginal, presignedUrlOriginal, err := s.GenerateUploadUrls(ctx, keyOriginal, item.MimeType)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	payload := models.ImagePayload{PublicURL: s.getRelativePath(publicUrlOriginal), Width: -1, Height: -1}
	payloadJson, err := json.Marshal(payload)
    if err != nil {
        return nil, apperrors.ErrInternal
    }

	NewItem := &models.Item{ItemData: models.ItemData{
		ID: itemId,
		RoomID: roomId,
		FolderID: item.FolderID,
		Title: item.Title,
		PreviewURL: s.getRelativePath(publicUrlPreview),
		SizeBytes: item.SizeBytes,
		ItemType: item.ItemType,
		MimeType: item.MimeType,
		Status: "uploading",
	},
		Payload: payloadJson,
	}

	err = s.repo.CreateItem(ctx, NewItem)
	if err != nil {
		return nil, err
	}

	response := &responses.CreatingItemPhoto{ItemId: itemId,PresignedUrlPreview: presignedUrlPreview, PresignedUrlOriginal: presignedUrlOriginal}

	return response, nil
}

func (s *WorkshopService) CreateVideoItem(ctx context.Context, roomId uuid.UUID, item *requests.CreatingItemVideo) (*responses.CreatingItemVideo, error) {
	if item.FolderID != nil {
		inRoom, err := s.ValidateFolderAccess(ctx, *item.FolderID, roomId)
		if err != nil {
			return nil, err
		}
		if !inRoom {
			return nil, apperrors.ErrNotFound
		}
	}

	ext := s.getExtensionFromMime(item.MimeType)

	itemId := uuid.New()
	keyPreview := fmt.Sprintf("%s/%s/%s.%s", roomId, itemId, uuid.New(), "jpeg")
	keyOriginal := fmt.Sprintf("%s/%s/%s.%s", roomId, itemId, uuid.New(), ext)

	publicUrlPreview, presignedUrlPreview, err := s.GenerateUploadUrls(ctx, keyPreview, "image/jpeg")
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	publicUrlOriginal, presignedUrlOriginal, err := s.GenerateUploadUrls(ctx, keyOriginal, item.MimeType)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	payload := models.VideoPayload{PublicURL: s.getRelativePath(publicUrlOriginal), Duration: item.Duration}
	payloadJson, err := json.Marshal(payload)
    if err != nil {
        return nil, apperrors.ErrInternal
    }

	NewItem := &models.Item{ItemData: models.ItemData{
		ID: itemId,
		RoomID: roomId,
		FolderID: item.FolderID,
		Title: item.Title,
		PreviewURL: s.getRelativePath(publicUrlPreview),
		SizeBytes: item.SizeBytes,
		ItemType: item.ItemType,
		MimeType: item.MimeType,
		Status: "uploading",
	},
		Payload: payloadJson,
	}

	err = s.repo.CreateItem(ctx, NewItem)
	if err != nil {
		return nil, err
	}

	response := &responses.CreatingItemVideo{ItemId: itemId,PresignedUrlPreview: presignedUrlPreview, PresignedUrlOriginal: presignedUrlOriginal}

	return response, nil
}

func (s *WorkshopService) GetContentRoot(ctx context.Context, roomId uuid.UUID) (*responses.Content, error) {
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

	return &responses.Content{Folders: resultFolders, Items: make([]*models.Item, 0)}, nil
}

func (s *WorkshopService) GetFolders(ctx context.Context, folderID uuid.UUID) ([]*responses.FolderShow, error) {
    folders, err := s.repo.GetFolders(ctx, folderID)
    if err != nil {
        return nil, err
    }

    result := make([]*responses.FolderShow, 0, len(folders))

    for _, f := range folders {
        show := &responses.FolderShow{}
        result = append(result, show.FromModel(f))
    }

    return result, nil
}

func (s *WorkshopService) GetContentFolder(ctx context.Context, folderID uuid.UUID) (*responses.Content, error) {
    folders, err := s.repo.GetFolders(ctx, folderID)
    if err != nil {
        return nil, err
    }

    result := make([]*responses.FolderShow, 0, len(folders))

    for _, f := range folders {
        show := &responses.FolderShow{}
        result = append(result, show.FromModel(f))
    }

    return &responses.Content{Folders: result, Items: make([]*models.Item, 0)}, nil
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

func (s *WorkshopService) GetRootFolders(ctx context.Context, roomId uuid.UUID) ([]*responses.FolderShow, error) {
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

	return resultFolders, nil
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

func (s *WorkshopService) UpdateStatusItem(ctx context.Context, roomId uuid.UUID, itemId uuid.UUID, status string) error {
	return s.repo.UpdateItemStatus(ctx, roomId, itemId, status)
}

func NewWorkshopService(repo *repositories.WorkshopRepository, s3 *s3.Client, s3pr *s3.PresignClient,
	bucketName string, endpoint string) *WorkshopService {
	return &WorkshopService{repo: repo, s3Client: s3, s3PresignedClient: s3pr, bucketName: bucketName,
	endpoint: endpoint}
}