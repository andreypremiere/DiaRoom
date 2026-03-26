package repositories

import (
	"context"
	"fmt"
	"post-microservice/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepositoryInter interface {
	GetCategoryIdBySlug(ctx context.Context, slug string) (int, error)
	CreatePost(ctx context.Context, data models.CreatePostInternal) (uuid.UUID, error)
	UpdatePostPreviewURL(ctx context.Context, postID uuid.UUID, previewURL string) error
	AddHashtagsToPost(ctx context.Context, postID uuid.UUID, hashtags []string) error
	InsertCanvasAndUpdatePost(ctx context.Context, postID uuid.UUID, payloadJSON []byte) error
}

type PostRepository struct {
	db *pgxpool.Pool	

}

func (r *PostRepository) UpdatePostPreviewURL(ctx context.Context, postID uuid.UUID, previewURL string) error {
	_, err := r.db.Exec(ctx, `
        UPDATE posts 
        SET preview_url = $1,
            updated_at = NOW()
        WHERE id = $2
    `, previewURL, postID)
	return err
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

func (r *PostRepository) CreatePost(ctx context.Context, data models.CreatePostInternal) (uuid.UUID, error) {
    var postID uuid.UUID
    err := r.db.QueryRow(ctx, `
        INSERT INTO posts (
            room_id, category_id, title, status, ai_check_status
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, data.RoomID, data.CategoryID, data.Title, data.Status, data.AiStatus).Scan(&postID)

	fmt.Println("Ошибка при создании поста: ", err)
	
    return postID, err
}

func (r *PostRepository) AddHashtagsToPost(ctx context.Context, postID uuid.UUID, hashtags []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Добавляем/обновляем хэштеги и получаем их ID
	rows, err := tx.Query(ctx, `
    INSERT INTO hashtags (name)
    SELECT unnest($1::text[])
    ON CONFLICT (name) 
    DO UPDATE SET 
        usage_count = hashtags.usage_count + 1
    RETURNING id
`, hashtags)
	if err != nil {
		return fmt.Errorf("failed to upsert hashtags: %w", err)
	}

	var hashtagIDs []int
	// Очень важно закрыть rows, используй defer или закрой вручную после цикла
	defer rows.Close() 

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan hashtag id: %w", err)
		}
		hashtagIDs = append(hashtagIDs, id)
	}

	// Проверка на ошибки после итерации
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %w", err)
	}
	rows.Close() // Закрываем до выполнения следующего шага

	// 2. Привязка (остается без изменений)
	if len(hashtagIDs) > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO posts_hashtags (post_id, hashtag_id)
			SELECT $1, unnest($2::int[])
			ON CONFLICT DO NOTHING
		`, postID, hashtagIDs)
		if err != nil {
			return fmt.Errorf("failed to link hashtags to post: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostRepository) InsertCanvasAndUpdatePost(ctx context.Context, postID uuid.UUID, payloadJSON []byte) error {
    tx, err := r.db.Begin(ctx) // Убедитесь, что r.db это *pgxpool.Pool
    if err != nil {
        return fmt.Errorf("failed to begin tx: %w", err)
    }
    defer tx.Rollback(ctx)

    var canvasID uuid.UUID

    // 1. Создаем Canvas. Используем string(payloadJSON) для надежности с типом JSONB
    insertCanvasQuery := `
        INSERT INTO canvases (payload) 
        VALUES ($1) 
        RETURNING id;
    `
    err = tx.QueryRow(ctx, insertCanvasQuery, string(payloadJSON)).Scan(&canvasID)
    if err != nil {
        return fmt.Errorf("failed to insert canvas: %w", err)
    }

    // 2. Обновляем Post
    updatePostQuery := `
        UPDATE posts 
        SET canvas_id = $1, updated_at = CURRENT_TIMESTAMP 
        WHERE id = $2;
    `
    tag, err := tx.Exec(ctx, updatePostQuery, canvasID, postID)
    if err != nil {
        return fmt.Errorf("failed to update post: %w", err)
    }

    // Проверка для pgx v5
    if tag.RowsAffected() == 0 {
        return fmt.Errorf("post not found")
    }

    return tx.Commit(ctx)
}

func NewPostRepository(db *pgxpool.Pool) *PostRepository {
	// Принимать базы данных какие-нибудь

	return &PostRepository{db: db}
}