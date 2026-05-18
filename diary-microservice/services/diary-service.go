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

	tags, err := s.repo.GetTagsByMessageIDs(ctx, msgIDs)
    if err != nil {
        return nil, err
    }

	result := &responses.GettingMessages{
		Messages: make([]*responses.MessageResponse, len(messages)),
	}

	for i, msg := range messages {
		msgTags := tags[msg.ID]
        if msgTags == nil {
            msgTags = []*models.Tag{}
        }

		result.Messages[i] = &responses.MessageResponse{
			Message:    msg,
			Attachment: attMap[msg.ID], 
			Tags: msgTags,
		}
	}

	return result, nil
}

func (s *DiaryService) CreateMessage(ctx context.Context, roomId uuid.UUID, req *requests.MessageCreateRequest) (*responses.MessageCreateResponse, error) {
	if req.Tags == nil {
		req.Tags = make([]*models.Tag, 0)
	}

	// Проверяем, что каждый добавляемый тег создается для той же комнаты
	for _, tag := range req.Tags {
		if tag.RoomId != roomId {
			return nil, apperrors.ErrAccess
		}
	}

	switch req.MsgType {
    case "standard":
        if len(req.Attachments) > 7 {
            return nil, apperrors.ErrInvalidInput // Слишком много файлов
        }
    case "voice_note", "video_note":
        if len(req.Attachments) != 1 {
            return nil, apperrors.ErrInvalidInput // Должен быть строго один файл
        }
    default:
        return nil, apperrors.ErrInvalidInput
    }

    // Инициализация общих данных
    messageUUID := uuid.New()
    status := "sending"

    newMessage := &models.Message{
        ID:                       messageUUID,
        RoomID:                   roomId,
        MsgType:                  req.MsgType,
        Content:                  req.Content,
        Status:                  &status,
        AttachedObjectWorkshopID: req.WorkshopFolderId,
        AttachedObjectPostID:     req.PublicationPostId,
    }

    attachments := make([]*models.Attachment, 0, len(req.Attachments))
    uploadItems := make([]*responses.AttachmentUploadItem, 0, len(req.Attachments))

    //Общий цикл обработки вложений
    for _, item := range req.Attachments {
        attachId := uuid.New()
        
        // Генерация основного файла
        ext := s.s3.getExtensionFromMimeType(item.MimeType)
        key := fmt.Sprintf("%s/%s/%s%s", roomId, messageUUID, uuid.New(), ext)
        _, presignedUrl, err := s.s3.GenerateUploadUrls(ctx, key, item.MimeType)
        if err != nil {
            return nil, apperrors.ErrInternal
        }

        newAttach := &models.Attachment{
            ID:            attachId,
            RoomID:        roomId,
            MessageID:     messageUUID,
            AttType:       item.AttType,
            S3Key:         key,
            FileSizeBytes: item.FileSizeBytes,
            Duration:      item.Duration,
        }

        attachResponse := &responses.AttachmentUploadItem{
            AttachmentID: attachId,
            PresignedURL: presignedUrl,
        }

        // Логика превью (только для видео-заметок или стандартных медиа)
        if req.MsgType == "video_note" || req.MsgType == "standard" {
            keyPreview := fmt.Sprintf("%s/%s/%s.jpeg", roomId, messageUUID, uuid.New())
            _, presignedUrlPreview, err := s.s3.GenerateUploadUrls(ctx, keyPreview, "image/jpeg")
            if err != nil {
                return nil, apperrors.ErrInternal
            }
            newAttach.PreviewS3Key = &keyPreview
            attachResponse.PresignedPreviewURL = &presignedUrlPreview
        }

        attachments = append(attachments, newAttach)
        uploadItems = append(uploadItems, attachResponse)
    }

    // Сохранение в БД
    if err := s.repo.CreateMessageWithAttachments(ctx, newMessage, attachments, req.Tags); err != nil {
        return nil, err
    }

    return &responses.MessageCreateResponse{
        MessageID:   messageUUID,
        Status:      status,
        UploadItems: uploadItems,
    }, nil
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

	// Получаем теги для сообщения
	tags, err := s.repo.GetTagsByMessageIDs(ctx, []uuid.UUID{msg.ID})
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

	msgTags := tags[msg.ID]
	if msgTags == nil {
		msgTags = []*models.Tag{}
	}
	
    // Упаковываем в итоговую структуру
    return &responses.UpdatingStatus{
        Message: &responses.MessageResponse{
            Message:    msg,
            Attachment: attachments,
			Tags: msgTags,
        },
    }, nil
}

func (s *DiaryService) CreateTag(ctx context.Context, req *requests.CreatingTag, roomId uuid.UUID) (*models.Tag, error) {
	newTag := &models.Tag{Id: uuid.New(), RoomId: roomId, Name: req.Name, Color: req.Color}

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

func (s *DiaryService) DeleteMessage(ctx context.Context, roomId uuid.UUID, messageId uuid.UUID) error {
	// Проверка, что такое сообщение с таким roomId существует
	message, err := s.repo.GetMessageByID(ctx, roomId, messageId)
	if err != nil {
		return err
	}

	// Собщение найдено
	if message == nil {
		return apperrors.ErrNotFound
	}

	// Ищем все вложения
	attachments, err := s.repo.GetAttachmentsByMessageIDs(ctx, []uuid.UUID{message.ID})
	if err != nil {
		return err
	}

	if len(attachments) > 0 {
		deletingKeys := make([]string, 0)

		for _, item := range attachments {
			if item.S3Key != "" {
				deletingKeys = append(deletingKeys, item.S3Key)
			}
			if item.PreviewS3Key != nil && *item.PreviewS3Key != "" {
				deletingKeys = append(deletingKeys, *item.PreviewS3Key)
			}
		}

		err = s.s3.DeleteByKeys(ctx, deletingKeys) 
		if err != nil {
			return apperrors.ErrInternal
		}
	}

	// Удаляем сообщение
	err = s.repo.DeleteMessage(ctx, roomId, messageId)

	return err
}

func NewDiaryService(repo *repositories.DiaryRepository, s3 *S3Manager, redisClient *redis.Client) *DiaryService {
	return &DiaryService{
		repo:  repo,
		s3:    s3,
		redis: redisClient,
	}
}