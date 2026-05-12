package repositories

import (
	"context"
	apperrors "diary-microservice/app-errors"
	"diary-microservice/models"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiaryRepository struct {
	db *pgxpool.Pool
}

func (r *DiaryRepository) parseError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": 
			return apperrors.ErrAlreadyExists
		case "23503": 
			return apperrors.ErrNotFound
		case "22P02": 
			return apperrors.ErrInvalidInput
		}
	}

	return apperrors.ErrInternal
}

func (r *DiaryRepository) CreateMessageWithAttachments(ctx context.Context, msg *models.Message, attachments []*models.Attachment) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return apperrors.ErrInternal
	}

	defer tx.Rollback(ctx)

	// Вставляем сообщение
	queryMsg := `
		INSERT INTO messages (
			id, room_id, msg_type, content, status, 
			attached_object_workshop_id, attached_object_post_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.Exec(ctx, queryMsg,
		msg.ID,
		msg.RoomID,
		msg.MsgType,
		msg.Content,
		msg.Status,
		msg.AttachedObjectWorkshopID,
		msg.AttachedObjectPostID,
	)
	if err != nil {
		return r.parseError(err)
	}

	// Вставляем вложения (если они есть)
	if len(attachments) > 0 {
		// Используем пакетную вставку для производительности
		queryAttr := `
			INSERT INTO attachments (
				id, room_id, message_id, att_type, s3_key, 
				preview_s3_key, file_size_bytes, duration
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		
		for _, a := range attachments {
			if a == nil { continue } 
			
			_, err = tx.Exec(ctx, queryAttr,
				a.ID,
				a.RoomID,
				a.MessageID,
				a.AttType,
				a.S3Key,
				a.PreviewS3Key,
				a.FileSizeBytes,
				a.Duration,
			)
			if err != nil {
				return r.parseError(err)
			}
		}
	}

	//  Подтверждаем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return r.parseError(err)
	}

	return nil
}

func (r *DiaryRepository) UpdateMessageStatus(ctx context.Context, roomId uuid.UUID, messageId string, status string) error {
	query := `
		UPDATE messages 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND room_id = $3
	`

	result, err := r.db.Exec(ctx, query, status, messageId, roomId)
	if err != nil {
		return r.parseError(err)
	}


	if result.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

func NewDiaryRepository(db *pgxpool.Pool) *DiaryRepository {
	return &DiaryRepository{
		db: db,
	}
}