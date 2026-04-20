package repositories

import (
	"context"
	"errors"
	"fmt"
	apperrors "post-microservice/app-errors"
	"post-microservice/contracts/responses"
	"post-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *PostRepository) parseError(err error) error {
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

	if errors.Is(err, redis.Nil) {
		return apperrors.ErrNotFound
	}

	return apperrors.ErrInternal
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
		return nil, r.parseError(err)
	}

	return &post, nil
}

func (r *PostRepository) GetAllPosts(ctx context.Context) ([]responses.PostInfo, error) {
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
		return nil, r.parseError(err)
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
			return nil, r.parseError(err)
		}
		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, r.parseError(err)
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
	return r.parseError(err)
}

func (r *PostRepository) PushPostToQueue(ctx context.Context, postID uuid.UUID) error {
    err := r.redis.LPush(ctx, "new_posts:post_id", postID.String()).Err()
    if err != nil {
        return r.parseError(err)
    }
    return nil
}

func (r *PostRepository) GetCategoryIdBySlug(ctx context.Context, slug string) (int, error) {
    var id int
    
    query := `SELECT id FROM categories WHERE slug = $1 LIMIT 1`
    
    err := r.db.QueryRow(ctx, query, slug).Scan(&id)
    if err != nil {
        return 0, r.parseError(err)
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

	if err != nil {
		return postID, r.parseError(err)
	}
	
    return postID, err
}

func (r *PostRepository) AddHashtagsToPost(ctx context.Context, postID uuid.UUID, hashtags []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return r.parseError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
    INSERT INTO hashtags (name)
    SELECT unnest($1::text[])
    ON CONFLICT (name) 
    DO UPDATE SET 
        usage_count = hashtags.usage_count + 1
    RETURNING id
`, hashtags)
	if err != nil {
		return r.parseError(err)
	}

	var hashtagIDs []int
	defer rows.Close() 

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return r.parseError(err)
		}
		hashtagIDs = append(hashtagIDs, id)
	}

	if err := rows.Err(); err != nil {
		return r.parseError(err)
	}
	rows.Close() 

	if len(hashtagIDs) > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO posts_hashtags (post_id, hashtag_id)
			SELECT $1, unnest($2::int[])
			ON CONFLICT DO NOTHING
		`, postID, hashtagIDs)
		if err != nil {
			return r.parseError(err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return r.parseError(err)
	}

	return err
}

func (r *PostRepository) InsertCanvasAndUpdatePost(ctx context.Context, postID uuid.UUID, payloadJSON []byte) error {
    tx, err := r.db.Begin(ctx) 
    if err != nil {
        return r.parseError(err)
    }
    defer tx.Rollback(ctx)

    var canvasID uuid.UUID

    insertCanvasQuery := `
        INSERT INTO canvases (payload) 
        VALUES ($1) 
        RETURNING id;
    `
    err = tx.QueryRow(ctx, insertCanvasQuery, string(payloadJSON)).Scan(&canvasID)
    if err != nil {
        return r.parseError(err)
    }

    updatePostQuery := `
        UPDATE posts 
        SET canvas_id = $1, updated_at = CURRENT_TIMESTAMP 
        WHERE id = $2;
    `
    tag, err := tx.Exec(ctx, updatePostQuery, canvasID, postID)
    if err != nil {
        return r.parseError(err)
    }

    if tag.RowsAffected() == 0 {
        return fmt.Errorf("post not found")
    }

    err = tx.Commit(ctx)
	if err != nil {
		return r.parseError(err)
	}

	return err
}

func NewPostRepository(db *pgxpool.Pool, redis *redis.Client) *PostRepository {
	return &PostRepository{db: db, redis: redis}
}