package repositories

import (
	"context"
	"errors"
	apperrors "workshop-microservice/app-errors"
	"workshop-microservice/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkshopRepository struct {
	db *pgxpool.Pool
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

func NewWorkshopRepository(db *pgxpool.Pool) *WorkshopRepository {
	return &WorkshopRepository{db: db}
}