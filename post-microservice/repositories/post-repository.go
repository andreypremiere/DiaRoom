package repositories

import (
	"context"
	"fmt"
	"post-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepositoryInter interface {
	GetCategoryIdBySlug(ctx context.Context, slug string) (int, error)
	SavePost(ctx context.Context, req models.CreatePostRequest, categoryId int) (uuid.UUID, error)
	PublishPost(ctx context.Context, req models.PublishPostRequest) error
}

type PostRepository struct {
	db *pgxpool.Pool	

}


func (r *PostRepository) GetCategoryIdBySlug(ctx context.Context, slug string) (int, error) {
    var id int
    
    query := `SELECT id FROM categories WHERE slug = $1 LIMIT 1`
    
    // Используем QueryRowContext, так как ожидаем ровно одно значение
    err := r.db.QueryRow(ctx, query, slug).Scan(&id)
    if err != nil {
        // У pgx свои ошибки, но логика та же
        return 0, fmt.Errorf("failed to get category: %w", err)
    }

    return id, nil
}


func (r *PostRepository) SavePost(ctx context.Context, req models.CreatePostRequest, categoryId int) (uuid.UUID, error) {
	var postId uuid.UUID

	queryPost := `INSERT INTO posts (room_id, category_id, title) 
                  VALUES ($1, $2, $3) RETURNING id`
	err := r.db.QueryRow(ctx, queryPost, req.RoomID, categoryId, req.Title).Scan(&postId)

	if err != nil {
		return uuid.Nil, fmt.Errorf("insert post: %w", err)
	}

	return postId, nil
}

func (r *PostRepository) PublishPost(ctx context.Context, req models.PublishPostRequest) error {
	// Начинаем транзакцию
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Создаем Canvas и получаем его ID
	canvasID, err := r.createCanvasTx(ctx, tx, req.Payload)
	if err != nil {
		return fmt.Errorf("failed to create canvas: %w", err)
	}

	// 2. Обновляем основную таблицу Post
	err = r.updatePostTx(ctx, tx, req.PostID, canvasID, req.PreviewURL)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	// 3. Обрабатываем хештеги (если они есть)
	if len(req.Hashtags) > 0 {
		err = r.upsertHashtagsTx(ctx, tx, req.PostID, req.Hashtags)
		if err != nil {
			return fmt.Errorf("failed to link hashtags: %w", err)
		}
	}

	// Подтверждаем транзакцию
	return tx.Commit(ctx)
}

// --- Приватные методы для логического разделения операций ---

func (r *PostRepository) createCanvasTx(ctx context.Context, tx pgx.Tx, payload []byte) (uuid.UUID, error) {
	var canvasID uuid.UUID
	query := `INSERT INTO canvases (payload) VALUES ($1) RETURNING id`
	
	err := tx.QueryRow(ctx, query, payload).Scan(&canvasID)
	return canvasID, err
}

func (r *PostRepository) updatePostTx(ctx context.Context, tx pgx.Tx, postID, canvasID uuid.UUID, previewURL *string) error {
	// COALESCE позволяет не затирать старый preview_url, если новый не передан (nil)
	query := `
		UPDATE posts 
		SET canvas_id = $1,
		    preview_url = COALESCE($2, preview_url),
		    published_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`
	
	commandTag, err := tx.Exec(ctx, query, canvasID, previewURL, postID)
	if err != nil {
		return err
	}
	
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("post not found or already deleted")
	}
	
	return nil
}

func (r *PostRepository) upsertHashtagsTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, tags []string) error {
	// Идеальный подход для Highload: используем unnest() для массовой вставки (Bulk Insert).
	// Это выполнит вставку/обновление всех хештегов за один поход в БД.
	query := `
		WITH new_tags AS (
			INSERT INTO hashtags (name, usage_count)
			SELECT unnest($1::text[]), 1
			ON CONFLICT (name) DO UPDATE 
			SET usage_count = hashtags.usage_count + 1
			RETURNING id
		)
		INSERT INTO posts_hashtags (post_id, hashtag_id)
		SELECT $2, id FROM new_tags
		ON CONFLICT DO NOTHING; -- Защита от дублирования связи
	`
	
	_, err := tx.Exec(ctx, query, tags, postID)
	return err
}

func NewPostRepository(db *pgxpool.Pool) *PostRepository {
	// Принимать базы данных какие-нибудь

	return &PostRepository{db: db}
}