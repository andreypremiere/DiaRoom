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

func (s *DiaryService) GetMessages(ctx context.Context, roomID uuid.UUID, limit, offset int) (*responses.GettingMessages, error) {
	messages, err := s.repo.GetMessagesByRoom(ctx, roomID, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return &responses.GettingMessages{Messages: []*responses.MessageResponse{}}, nil
	}

	msgIDs := make([]uuid.UUID, len(messages))
	for i, m := range messages {
		msgIDs[i] = m.ID
	}

	allAttachments, err := s.repo.GetAttachmentsByMessageIDs(ctx, msgIDs)
	if err != nil {
		return nil, err
	}

	if len(allAttachments) > 0 {
		for idx, _ := range allAttachments {
			if allAttachments[idx].PreviewS3Key != nil {
				fullURL := s.s3.getFullUrlFromKey(*allAttachments[idx].PreviewS3Key)
				allAttachments[idx].PreviewS3Key = &fullURL
			}

			fullURL := s.s3.getFullUrlFromKey(allAttachments[idx].S3Key)
			allAttachments[idx].S3Key = fullURL
		}
	}

	attMap := make(map[uuid.UUID][]*models.Attachment)
	for _, att := range allAttachments {
		attMap[att.MessageID] = append(attMap[att.MessageID], att)
	}

	result := &responses.GettingMessages{
		Messages: make([]*responses.MessageResponse, len(messages)),
	}

	for i, msg := range messages {
		result.Messages[i] = &responses.MessageResponse{
			Message:    msg,
			Attachment: attMap[msg.ID], 
		}
	}

	return result, nil
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

	} else if req.MsgType == "voice_note" {
		if len(req.Attachments) != 1 {
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
				FileSizeBytes: item.FileSizeBytes,
				Duration: item.Duration,
			}

			attachments = append(attachments, newAttach)

			attachResponse := &responses.AttachmentUploadItem{
				AttachmentID: attachId,
				PresignedURL: presignedUrl,
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
	} else if req.MsgType == "video_note" {
		if len(req.Attachments) != 1 {
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
			// Получаем публичный и статичный ключ для файла
			key := fmt.Sprintf("%s/%s/%s%s", roomId, messageUUID, uuid.New(), s.s3.getExtensionFromMimeType(item.MimeType))
			_, presignedUrl, err := s.s3.GenerateUploadUrls(ctx, key, item.MimeType)
			if err != nil {
				return nil, apperrors.ErrInternal
			}
			// Получаем публичный и статичный ключ для превью
			keyPreview := fmt.Sprintf("%s/%s/%s%s", roomId, messageUUID, uuid.New(), ".jpeg")
			_, presignedUrlPreview, err := s.s3.GenerateUploadUrls(ctx, keyPreview, "image/jpeg")
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

func (s *DiaryService) UpdateMessageStatus(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingMessage) (*responses.UpdatingStatus, error) {
	// Обновляем статус
	err := s.repo.UpdateMessageStatus(ctx, roomId, req.MessageID, req.Status)
    if err != nil {
        return nil, err
    }

	// Получаем сообщение
    msg, err := s.repo.GetMessageByID(ctx, roomId, req.MessageID)
    if err != nil {
        return nil, err
    }

    // Получаем вложения для сообщения
    attachments, err := s.repo.GetAttachmentsByMessageIDs(ctx, []uuid.UUID{msg.ID})
    if err != nil {
        return nil, err
    }

	// Делаем полные ссылки s3
	for idx, _ := range attachments {
			if attachments[idx].PreviewS3Key != nil {
				fullURL := s.s3.getFullUrlFromKey(*attachments[idx].PreviewS3Key)
				attachments[idx].PreviewS3Key = &fullURL
			}

			fullURL := s.s3.getFullUrlFromKey(attachments[idx].S3Key)
			attachments[idx].S3Key = fullURL
		}

    // Упаковываем в итоговую структуру
    return &responses.UpdatingStatus{
        Message: &responses.MessageResponse{
            Message:    msg,
            Attachment: attachments,
        },
    }, nil
}

func (s *DiaryService) CreateTag(ctx context.Context, req *requests.CreatingTag, roomId uuid.UUID) (*models.Tag, error) {
	newTag := models.FromCreatingTag(req, roomId, uuid.New())

	err := s.repo.CreateTag(ctx, newTag)
	if err != nil {
		return nil, err
	}

	return newTag, nil 
}

func (s *DiaryService) UpdateTag(ctx context.Context, req *requests.UpdatingTag, tagId uuid.UUID, roomId uuid.UUID) (*models.Tag, error) {
    tag := &models.Tag{
        Id:     tagId,
        RoomId: roomId,
        Name:   req.Name,
        Color:  req.Color,
    }

    err := s.repo.UpdateTag(ctx, tag)
    if err != nil {
        return nil, err
    }

    return tag, nil
}

func (s *DiaryService) DeleteTag(ctx context.Context, tagId uuid.UUID, roomId uuid.UUID) error {
    err := s.repo.DeleteTag(ctx, tagId, roomId)
    if err != nil {
        return err
    }

    return nil
}

func (s *DiaryService) GetTags(ctx context.Context, roomId uuid.UUID) ([]*models.Tag, error) {
	tags, err := s.repo.GetTags(ctx, roomId)
	if err != nil {
		return nil, err
	}
	
	return tags, nil
}

func NewDiaryService(repo *repositories.DiaryRepository, s3 *S3Manager, redisClient *redis.Client) *DiaryService {
	return &DiaryService{
		repo:  repo,
		s3:    s3,
		redis: redisClient,
	}
}