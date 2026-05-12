package services

import (
	"context"
	apperrors "diary-microservice/app-errors"
	"diary-microservice/contracts/requests"
	"diary-microservice/contracts/responses"
	"diary-microservice/models"
	"diary-microservice/repositories"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DiaryService struct {
	repo  *repositories.DiaryRepository
	s3    *S3Manager
	redis *redis.Client
}

func (s *DiaryService) CreateMessage(ctx context.Context, roomId uuid.UUID, req *requests.MessageCreateRequest) (*responses.MessageCreateResponse, error) {
	// Обработка для стандартного сообщения
	if req.MsgType == "standard" {
		if len(req.Attachments) > 7 {
			return nil, apperrors.ErrInternal
		}

		// Создаем новый объект сообщения
		messageUUID := uuid.New()
		status := "sending"

		newMessage := &models.Message{
			ID: messageUUID,
			RoomID: roomId,
			MsgType: req.MsgType,
			Content: req.Content,
			Status: &status,
			AttachedObjectWorkshopID: req.WorkshopFolderId,
			AttachedObjectPostID: req.PublicationPostId,
		}

		// Формируем список ответов
		attachmentsCreating := make([]*responses.AttachmentUploadItem, 0, len(req.Attachments))

		// Создаем список вложений
		attachments := make([]*models.Attachment, 0, len(req.Attachments))

		for _, item := range req.Attachments {

			//Получаем публичный и статичный ключ для превью (roomId/messageId/uuid.ext)
			keyPreview := fmt.Sprintf("%s/%s/%s%s", roomId, messageUUID, uuid.New(), ".jpeg")
			_, presignedUrlPreview, err := s.s3.GenerateUploadUrls(ctx, keyPreview, "image/jpeg")
			if err != nil {
				return nil, apperrors.ErrInternal
			}

			// Получаем публичный и статичный ключ для файла
			key := fmt.Sprintf("%s/%s/%s%s", roomId, messageUUID, uuid.New(), s.s3.getExtensionFromMimeType(item.MimeType))
			_, presignedUrl, err := s.s3.GenerateUploadUrls(ctx, key, item.MimeType)
			if err != nil {
				return nil, apperrors.ErrInternal
			}
			attachId := uuid.New()
			newAttach := &models.Attachment{
				ID: attachId,
				RoomID: roomId,
				MessageID: messageUUID,
				AttType: item.AttType,
				S3Key: key,
				PreviewS3Key: &keyPreview,
				FileSizeBytes: item.FileSizeBytes,
				Duration: item.Duration,
			}

			attachments = append(attachments, newAttach)

			attachResponse := &responses.AttachmentUploadItem{
				AttachmentID: attachId,
				PresignedURL: presignedUrl,
				PresignedPreviewURL: &presignedUrlPreview,
			}

			attachmentsCreating = append(attachmentsCreating, attachResponse)
		}

		err := s.repo.CreateMessageWithAttachments(ctx, newMessage, attachments)
		if err != nil {
			return nil, err
		}

		messageResponse := &responses.MessageCreateResponse{
			MessageID: messageUUID,
			Status: status,
			UploadItems: attachmentsCreating,
		}

		return messageResponse, nil

	} else {
		return nil, apperrors.ErrInvalidInput
	}
}

func (s *DiaryService) UpdateMessageStatus(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingMessage) error {
	return s.repo.UpdateMessageStatus(ctx, roomId, req.MessageID, req.Status)
}

func NewDiaryService(repo *repositories.DiaryRepository, s3 *S3Manager, redisClient *redis.Client) *DiaryService {
	return &DiaryService{
		repo:  repo,
		s3:    s3,
		redis: redisClient,
	}
}