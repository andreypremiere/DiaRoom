package repositories

import (
	"context"
	"fmt"
	"post-microservice/contracts/responses"
	"post-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type PostRepositoryInter interface {
	GetCategoryIdBySlug(ctx context.Context, slug string) (int, error)
	CreatePost(ctx context.Context, data models.CreatePostInternal) (uuid.UUID, error)
	UpdatePostPreviewURL(ctx context.Context, postID uuid.UUID, previewURL string) error
	AddHashtagsToPost(ctx context.Context, postID uuid.UUID, hashtags []string) error
	InsertCanvasAndUpdatePost(ctx context.Context, postID uuid.UUID, payloadJSON []byte) error
    PushPostToQueue(ctx context.Context, postID uuid.UUID) error
    GetAllPosts(ctx context.Context) ([]responses.PostInfo, error)
    GetPostForShowing(ctx context.Context, postID uuid.UUID) (*responses.ShowingPost, error)
}

type PostRepository struct {
	db *pgxpool.Pool	
    redis *redis.Client

}

func (r *PostRepository) GetPostForShowing(ctx context.Context, postID uuid.UUID) (*responses.ShowingPost, error) {
	query := `
		SELECT 
			p.room_id, 
			c.payload, 
			cat.slug, 
			p.views_count, 
			p.likes_count,
			COALESCE(
				(SELECT array_agg(h.name) 
				 FROM posts_hashtags ph 
				 JOIN hashtags h ON ph.hashtag_id = h.id 
				 WHERE ph.post_id = p.id), 
				'{}'
			) as hashtags
		FROM posts p
		INNER JOIN canvases c ON p.canvas_id = c.id
		INNER JOIN categories cat ON p.category_id = cat.id
		WHERE p.id = $1 AND p.is_deleted = FALSE
	`

	var post responses.ShowingPost
	
	err := r.db.QueryRow(ctx, query, postID).Scan(
		&post.RoomId,
		&post.Canvas,
		&post.Category,
		&post.ViewsCount,
		&post.LikesCount,
		&post.Hashtags,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch post: %w", err)
	}

	return &post, nil
}

func (r *PostRepository) GetAllPosts(ctx context.Context) ([]responses.PostInfo, error) {
	// SQL запрос с JOIN для получения slug категории
	query := `
		SELECT 
			p.id, 
			p.room_id, 
			p.preview_url, 
			c.slug as category_slug, 
			p.canvas_id, 
			p.title, 
			p.views_count, 
			p.likes_count
		FROM posts p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.status = 'published' 
		  AND p.is_deleted = FALSE
          AND p.canvas_id IS NOT NULL
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var posts []responses.PostInfo

	for rows.Next() {
		var p responses.PostInfo
		err := rows.Scan(
			&p.Id,
			&p.RoomId,
			&p.PreviewUrl,
			&p.Category,
			&p.CanvasId,
			&p.Title,
			&p.ViewsCount,
			&p.LikesCount,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
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

// PushPostToQueue кладет ID поста в список Redis
func (r *PostRepository) PushPostToQueue(ctx context.Context, postID uuid.UUID) error {
    // Используем LPUSH для добавления в начало списка
    // Ключ лучше вынести в константу, например "posts:queue:new"
    err := r.redis.LPush(ctx, "new_posts:post_id", postID.String()).Err()
    if err != nil {
        return fmt.Errorf("failed to push post to redis: %w", err)
    }
    return nil
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

func NewPostRepository(db *pgxpool.Pool, redis *redis.Client) *PostRepository {
	// Принимать базы данных какие-нибудь

	return &PostRepository{db: db, redis: redis}
}