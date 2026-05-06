package repositories

import (
	"context"
	"errors"
	apperrors "workshop-microservice/app-errors"
	"workshop-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkshopRepository struct {
	db *pgxpool.Pool
}

func (r *WorkshopRepository) GetItems(ctx context.Context, roomID uuid.UUID, folderID *uuid.UUID) ([]*models.Item, error) {
    query := `
        SELECT 
            id, room_id, folder_id, title, preview_url, 
            size_bytes, item_type, status, mime_type, 
            created_at, updated_at, payload
        FROM items
        WHERE room_id = $1 
          AND (folder_id = $2 OR (folder_id IS NULL AND $2 IS NULL))
        ORDER BY created_at DESC
    `

    rows, err := r.db.Query(ctx, query, roomID, folderID)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    var items []*models.Item
    for rows.Next() {
        var item models.Item
        err := rows.Scan(
            &item.ItemData.ID,
            &item.ItemData.RoomID,
            &item.ItemData.FolderID,
            &item.ItemData.Title,
            &item.ItemData.PreviewURL,
            &item.ItemData.SizeBytes,
            &item.ItemData.ItemType,
            &item.ItemData.Status,
            &item.ItemData.MimeType,
            &item.ItemData.CreatedAt,
            &item.ItemData.UpdatedAt,
            &item.Payload, 
        )
        if err != nil {
            return nil, r.parseError(err)
        }
        items = append(items, &item)
    }

    if err = rows.Err(); err != nil {
        return nil, r.parseError(err)
    }

    return items, nil
}

func (r *WorkshopRepository) parseError(err error) error {
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

func (r *WorkshopRepository) CreateItem(ctx context.Context, item *models.Item) error {
	query := `
		INSERT INTO items (
			id, room_id, folder_id, title, preview_url, 
			size_bytes, item_type, mime_type, status, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		item.ItemData.ID,
		item.ItemData.RoomID,
		item.ItemData.FolderID,
		item.ItemData.Title,
		item.ItemData.PreviewURL,
		item.ItemData.SizeBytes,
		item.ItemData.ItemType,
		item.ItemData.MimeType,
		item.ItemData.Status,
		item.Payload,
	)

	if err != nil {
		return r.parseError(err)
	}

	return nil
}

func (r *WorkshopRepository) IsFolderInRoom(ctx context.Context, folderID uuid.UUID, roomID uuid.UUID) (bool, error) {
    var exists bool
    query := `SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1 AND room_id = $2)`
    
    err := r.db.QueryRow(ctx, query, folderID, roomID).Scan(&exists)
    if err != nil {
        return false, apperrors.ErrNotFound
    }
    return exists, nil
}

func (r *WorkshopRepository) MoveFolder(ctx context.Context, roomID uuid.UUID, folderID uuid.UUID, newParentID *uuid.UUID) error {

	var exists bool

	// Проверяем, что перемещаемая папка существует в комнате
	err := r.db.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM folders
			WHERE id = $1 AND room_id = $2
		)`,
		folderID,
		roomID,
	).Scan(&exists)

	if err != nil {
		return r.parseError(err)
	}

	if !exists {
		return apperrors.ErrNotFound
	}

	// Если указан новый родитель — проверяем его
	if newParentID != nil {

		err = r.db.QueryRow(
			ctx,
			`SELECT EXISTS(
				SELECT 1
				FROM folders
				WHERE id = $1 AND room_id = $2
			)`,
			*newParentID,
			roomID,
		).Scan(&exists)

		if err != nil {
			return r.parseError(err)
		}

		if !exists {
			return apperrors.ErrNotFound
		}

		// Нельзя переместить папку в саму себя
		if *newParentID == folderID {
			return apperrors.ErrInvalidInput
		}

		// Проверка на цикл:
		// нельзя переместить папку в своего потомка
		queryCycle := `
			WITH RECURSIVE descendants AS (
				SELECT id
				FROM folders
				WHERE id = $1

				UNION ALL

				SELECT f.id
				FROM folders f
				JOIN descendants d ON f.parent_id = d.id
			)
			SELECT EXISTS(
				SELECT 1
				FROM descendants
				WHERE id = $2
			);
		`

		var isCycle bool

		err = r.db.QueryRow(
			ctx,
			queryCycle,
			folderID,
			*newParentID,
		).Scan(&isCycle)

		if err != nil {
			return r.parseError(err)
		}

		if isCycle {
			return apperrors.ErrInvalidInput
		}
	}

	// Выполняем перемещение
	// если newParentID == nil -> parent_id = NULL
	result, err := r.db.Exec(
		ctx,
		`
		UPDATE folders
		SET parent_id = $1,
		    updated_at = NOW()
		WHERE id = $2
		  AND room_id = $3
		`,
		newParentID,
		folderID,
		roomID,
	)

	if err != nil {
		return r.parseError(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

func (r *WorkshopRepository) RenameFolder(ctx context.Context, folderID, roomID uuid.UUID, newName string) error {
    query := `
        UPDATE folders 
        SET name = $1, updated_at = NOW() 
        WHERE id = $2 AND room_id = $3
    `

    result, err := r.db.Exec(ctx, query, newName, folderID, roomID)
    if err != nil {
        return r.parseError(err)
    }

    if result.RowsAffected() == 0 {
        return apperrors.ErrNotFound 
    }

    return nil
}

// Запрос только папок корня
func (r *WorkshopRepository) GetRootFolders(ctx context.Context, roomId uuid.UUID) ([]*models.Folder, error) {
	query := `
        SELECT id, room_id, parent_id, name, created_at, updated_at 
        FROM folders 
        WHERE room_id = $1 AND parent_id IS NULL
        ORDER BY created_at ASC;
    `

    rows, err := r.db.Query(ctx, query, roomId)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    folders := make([]*models.Folder, 0)

    for rows.Next() {
        var f models.Folder
        err := rows.Scan(
            &f.ID, 
            &f.RoomID, 
            &f.ParentID, 
            &f.Name, 
            &f.CreatedAt, 
            &f.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        folders = append(folders, &f)
    }

    if err := rows.Err(); err != nil {
        return nil, r.parseError(err)
    }

    return folders, nil
}

func (r *WorkshopRepository) CreateFolder(ctx context.Context, folder *models.Folder) (*models.Folder, error) {
    query := `
        INSERT INTO folders (room_id, parent_id, name)
        VALUES ($1, $2, $3)
        RETURNING id, room_id, parent_id, name, created_at, updated_at;
    `

    var createdFolder models.Folder

    err := r.db.QueryRow(ctx, query, 
        folder.RoomID, 
        folder.ParentID, 
        folder.Name,
    ).Scan(
        &createdFolder.ID,
        &createdFolder.RoomID,
        &createdFolder.ParentID,
        &createdFolder.Name,
        &createdFolder.CreatedAt,
        &createdFolder.UpdatedAt,
    )

    if err != nil {
        return nil, r.parseError(err) 
    }

    return &createdFolder, nil
}

// Запрос только папок по родителю
func (r *WorkshopRepository) GetFolders(ctx context.Context, folderID uuid.UUID) ([]*models.Folder, error) {
    query := `
        SELECT id, room_id, parent_id, name, created_at, updated_at 
        FROM folders 
        WHERE parent_id = $1
        ORDER BY name ASC;
    `

    rows, err := r.db.Query(ctx, query, folderID)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    var folders []*models.Folder
    for rows.Next() {
        var f models.Folder
        if err := rows.Scan(&f.ID, &f.RoomID, &f.ParentID, &f.Name, &f.CreatedAt, &f.UpdatedAt); err != nil {
            return nil, r.parseError(err)
        }
        folders = append(folders, &f)
    }

    return folders, rows.Err()
}

func (r *WorkshopRepository) UpdateItemStatus(ctx context.Context, roomId uuid.UUID, itemId uuid.UUID, status string) error {
	query := `
		UPDATE items 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND room_id = $3
	`

	result, err := r.db.Exec(ctx, query, status, itemId, roomId)
	if err != nil {
		return r.parseError(err)
	}

	if result.RowsAffected() == 0 {
		return r.parseError(err)
	}

	return nil
}

func NewWorkshopRepository(db *pgxpool.Pool) *WorkshopRepository {
	return &WorkshopRepository{db: db}
}