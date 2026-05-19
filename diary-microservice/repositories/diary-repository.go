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

func (r *DiaryRepository) GetMessagesByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*models.Message, error) {
	query := `
		SELECT id, room_id, msg_type, content, status, attached_object_workshop_id, attached_object_post_id, created_at
		FROM messages
		WHERE room_id = $1 AND status = 'sent'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		m := new(models.Message)
		err := rows.Scan(
			&m.ID, &m.RoomID, &m.MsgType, &m.Content, &m.Status,
			&m.AttachedObjectWorkshopID, &m.AttachedObjectPostID, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (r *DiaryRepository) GetMessageByID(ctx context.Context, roomID uuid.UUID, messageID uuid.UUID) (*models.Message, error) {
    query := `
        SELECT id, room_id, msg_type, content, status, attached_object_workshop_id, attached_object_post_id, created_at
        FROM messages
        WHERE id = $1 AND room_id = $2
    `
    m := new(models.Message)
    err := r.db.QueryRow(ctx, query, messageID, roomID).Scan(
        &m.ID, &m.RoomID, &m.MsgType, &m.Content, &m.Status,
        &m.AttachedObjectWorkshopID, &m.AttachedObjectPostID, &m.CreatedAt,
    )
    if err != nil {
        return nil, r.parseError(err)
    }
    return m, nil
}

func (r *DiaryRepository) GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) ([]*models.Attachment, error) {
	if len(messageIDs) == 0 {
		return []*models.Attachment{}, nil
	}

	query := `
		SELECT id, room_id, message_id, att_type, s3_key, preview_s3_key, file_size_bytes, duration
		FROM attachments
		WHERE message_id = ANY($1)
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, messageIDs)
	if err != nil {
		return nil, r.parseError(err)
	}
	defer rows.Close()

	var attachments []*models.Attachment
	for rows.Next() {
		a := new(models.Attachment)
		err := rows.Scan(
			&a.ID, &a.RoomID, &a.MessageID, &a.AttType, &a.S3Key,
			&a.PreviewS3Key, &a.FileSizeBytes, &a.Duration,
		)
		if err != nil {
			return nil, r.parseError(err)
		}
		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (r *DiaryRepository) GetTagsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error) {
    // 1. Защита от пустого слайса, чтобы не делать холостой запрос в базу
    if len(messageIDs) == 0 {
        return make(map[uuid.UUID][]*models.Tag), nil
    }

    // 2. SQL-запрос с JOIN таблицы связей и таблицы самих тегов
    query := `
        SELECT mt.message_id, t.id, t.room_id, t.name, t.color
        FROM message_tags mt
        JOIN tags t ON mt.tag_id = t.id
        WHERE mt.message_id = ANY($1)
        ORDER BY t.name ASC
    `

    rows, err := r.db.Query(ctx, query, messageIDs)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    // Инициализируем карту результатов
    result := make(map[uuid.UUID][]*models.Tag)

    // Сканируем строки
    for rows.Next() {
        var msgID uuid.UUID
        tag := &models.Tag{}

        err := rows.Scan(
            &msgID,     
            &tag.Id,
            &tag.RoomId,
            &tag.Name,
            &tag.Color,
        )
        if err != nil {
            return nil, r.parseError(err)
        }

        result[msgID] = append(result[msgID], tag)
    }

    if err = rows.Err(); err != nil {
        return nil, r.parseError(err)
    }

    return result, nil
}

func (r *DiaryRepository) CreateMessageWithAttachments(ctx context.Context, msg *models.Message, attachments []*models.Attachment, tags []*models.Tag) error {
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

	// Связываем теги
	if len(tags) > 0 {
        queryTags := `
            INSERT INTO message_tags (message_id, tag_id) 
            VALUES ($1, $2)
        `

        for _, t := range tags {
            if t == nil { continue }

            _, err = tx.Exec(ctx, queryTags, msg.ID, t.Id)
            if err != nil {
                return r.parseError(err) // Если упадет, defer tx.Rollback отменит всё
            }
        }
    }

	//  Подтверждаем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return r.parseError(err)
	}

	return nil
}

func (r *DiaryRepository) UpdateMessageStatus(ctx context.Context, roomId uuid.UUID, messageId uuid.UUID, status string) error {
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

func (r *DiaryRepository) CreateTag(ctx context.Context, newTag *models.Tag) error {
	query := `
		INSERT INTO tags (id, room_id, name, color)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(ctx, query,
		newTag.Id,
		newTag.RoomId,
		newTag.Name,
		newTag.Color,
	)

	if err != nil {
		return r.parseError(err)
	}

	return nil
}

func (r *DiaryRepository) UpdateTag(ctx context.Context, tag *models.Tag) error {
    query := `
        UPDATE tags 
        SET name = $1, color = $2 
        WHERE id = $3 AND room_id = $4
    `

    result, err := r.db.Exec(ctx, query,
        tag.Name,
        tag.Color,
        tag.Id,
        tag.RoomId,
    )

    if err != nil {
        return r.parseError(err)
    }

    if result.RowsAffected() == 0 {
        return apperrors.ErrNotFound
    }

    return nil
}

func (r *DiaryRepository) DeleteTag(ctx context.Context, tagId uuid.UUID, roomId uuid.UUID) error {
    query := `
        DELETE FROM tags 
        WHERE id = $1 AND room_id = $2
    `

    result, err := r.db.Exec(ctx, query, tagId, roomId)
    if err != nil {
        return r.parseError(err)
    }

    if result.RowsAffected() == 0 {
        return apperrors.ErrNotFound
    }

    return nil
}

func (r *DiaryRepository) GetTags(ctx context.Context, roomId uuid.UUID) ([]*models.Tag, error) {
	query := `
		SELECT id, room_id, name, color 
		FROM tags 
		WHERE room_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, roomId)
	if err != nil {
		return nil, r.parseError(err)
	}
	defer rows.Close()

	tags := []*models.Tag{}

	for rows.Next() {
		tag := &models.Tag{}
		err := rows.Scan(
			&tag.Id,
			&tag.RoomId,
			&tag.Name,
			&tag.Color,
		)
		if err != nil {
			return nil, r.parseError(err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, r.parseError(err)
	}

	return tags, nil
}

func (r *DiaryRepository) DeleteMessage(ctx context.Context, roomId uuid.UUID, messageId uuid.UUID) error {
    query := `
        DELETE FROM messages 
        WHERE id = $1 AND room_id = $2
    `

    result, err := r.db.Exec(ctx, query, messageId, roomId)
    if err != nil {
        return r.parseError(err)
    }

    if result.RowsAffected() == 0 {
        return apperrors.ErrNotFound
    }

    return nil
}

func (r *DiaryRepository) GetMessagesByTagSubstring(ctx context.Context, roomId uuid.UUID, namePart string, limit int, offset int) ([]*models.Message, error) {
    query := `
        SELECT DISTINCT m.id, m.room_id, m.msg_type, m.content, 
                        m.attached_object_workshop_id, m.attached_object_post_id,
                        m.created_at, m.updated_at
        FROM messages m
        JOIN message_tags mt ON m.id = mt.message_id
        JOIN tags t ON mt.tag_id = t.id
        WHERE m.room_id = $1 
          AND t.name ILIKE $2
        ORDER BY m.created_at DESC
		LIMIT $3 OFFSET $4
    `

    rows, err := r.db.Query(ctx, query, roomId, "%"+namePart+"%", limit, offset)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    messages := make([]*models.Message, 0)
    for rows.Next() {
        m := new(models.Message)
        err := rows.Scan(
            &m.ID,
            &m.RoomID,
            &m.MsgType,
            &m.Content,
            &m.AttachedObjectWorkshopID,
            &m.AttachedObjectPostID,
            &m.CreatedAt,
            &m.UpdatedAt,
        )
        if err != nil {
            return nil, r.parseError(err)
        }
        messages = append(messages, m)
    }

    return messages, nil
}

func (r *DiaryRepository) GetMessagesByContentSubstring(ctx context.Context, roomId uuid.UUID, textPart string, limit, offset int) ([]*models.Message, error) {
    query := `
        SELECT id, room_id, msg_type, content, 
               attached_object_workshop_id, attached_object_post_id,
               created_at, updated_at
        FROM messages
        WHERE room_id = $1 
          AND content ILIKE $2
        ORDER BY created_at DESC
        LIMIT $3 OFFSET $4
    `

    rows, err := r.db.Query(ctx, query, roomId, "%"+textPart+"%", limit, offset)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    messages := make([]*models.Message, 0)

    for rows.Next() {
        m := new(models.Message)
        err := rows.Scan(
            &m.ID,
            &m.RoomID,
            &m.MsgType,
            &m.Content,
            &m.AttachedObjectWorkshopID,
            &m.AttachedObjectPostID,
            &m.CreatedAt,
            &m.UpdatedAt,
        )
        if err != nil {
            return nil, r.parseError(err)
        }
        messages = append(messages, m)
    }

    if err = rows.Err(); err != nil {
        return nil, r.parseError(err)
    }

    return messages, nil
}

func NewDiaryRepository(db *pgxpool.Pool) *DiaryRepository {
	return &DiaryRepository{
		db: db,
	}
}