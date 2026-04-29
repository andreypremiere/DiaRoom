package repositories

import (
	"context"
	"errors"
	"fmt"
	apperrors "post-microservice/app-errors"
	"post-microservice/contracts/responses"
	"post-microservice/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type PostRepositoryInter interface {
	// GetCategoryIdBySlug(ctx context.Context, slug string) (int, error)
	CreatePost(ctx context.Context, data models.CreatePostInternal) (uuid.UUID, error)
	UpdatePostPreviewURL(ctx context.Context, postID uuid.UUID, previewURL string) error
	AddHashtagsToPost(ctx context.Context, postID uuid.UUID, hashtags []string) error
	InsertCanvasAndUpdatePost(ctx context.Context, postID uuid.UUID, payloadJSON []byte) error
    PushPostToQueue(ctx context.Context, postID uuid.UUID) error
    GetAllPosts(ctx context.Context) ([]responses.PostInfo, error)
    GetPostForShowing(ctx context.Context, postID uuid.UUID) (*responses.ShowingPost, error)
	UpdateStatusUploaded(ctx context.Context, postID uuid.UUID) error
	GetPersonalPosts(ctx context.Context, roomId uuid.UUID) ([]responses.PostInfoPersonal, error)
	GetRoomPosts(ctx context.Context, roomId uuid.UUID) ([]responses.PostInfo, error)
	GetAllViews(ctx context.Context) (map[string]string, error)
	BulkIncrementViews(ctx context.Context, views map[string]int) error
	CheckView(ctx context.Context, key string) (int64, error)
	SetView(ctx context.Context, key string, count string, time time.Duration)
	HIncrView(ctx context.Context, postId string) error
	AddLike(ctx context.Context, postId, roomId uuid.UUID) error
	RemoveLike(ctx context.Context, postId, roomId uuid.UUID) error
	CheckLikeStatus(ctx context.Context, postId, roomId uuid.UUID) (bool, error)
	GetPostLikerIds(ctx context.Context, postId uuid.UUID, limit, offset int) ([]uuid.UUID, error)
	DeletePost(ctx context.Context, postId uuid.UUID) error
}

type PostRepository struct {
	db *pgxpool.Pool	
    redis *redis.Client
	redisStats *redis.Client
}

func (r *PostRepository) DeletePost(ctx context.Context, postId uuid.UUID) error {
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return r.parseError(err)
    }
    defer tx.Rollback(ctx)

    // Получаем canvas_id и ID хештегов перед удалением
    var canvasId uuid.UUID
    var hashtagIds []int

    err = tx.QueryRow(ctx, `
        SELECT canvas_id, 
               COALESCE((SELECT array_agg(hashtag_id) FROM posts_hashtags WHERE post_id = $1), '{}')
        FROM posts WHERE id = $1
    `, postId).Scan(&canvasId, &hashtagIds)
    
    if err != nil {
        return r.parseError(err)
    }

    // Уменьшаем счетчик использования хештегов
    if len(hashtagIds) > 0 {
        _, err = tx.Exec(ctx, `
            UPDATE hashtags 
            SET usage_count = usage_count - 1 
            WHERE id = any($1)
        `, hashtagIds)
        if err != nil {
            return r.parseError(err)
        }
    }

    // Удаляем связи с хештегами (таблица posts_hashtags)
    _, err = tx.Exec(ctx, "DELETE FROM posts_hashtags WHERE post_id = $1", postId)
    if err != nil {
        return r.parseError(err)
    }

    // Удаляем лайки (таблица post_likes)
    _, err = tx.Exec(ctx, "DELETE FROM post_likes WHERE post_id = $1", postId)
    if err != nil {
        return r.parseError(err)
    }

    // Удаляем сам пост
    result, err := tx.Exec(ctx, "DELETE FROM posts WHERE id = $1", postId)
    if err != nil {
        return r.parseError(err)
    }
    if result.RowsAffected() == 0 {
        return apperrors.ErrNotFound
    }

    // Удаляем canvas (так как canvas_id был во внешней таблице, удаляем его последним)
    if canvasId != uuid.Nil {
        _, err = tx.Exec(ctx, "DELETE FROM canvases WHERE id = $1", canvasId)
        if err != nil {
            return r.parseError(err)
        }
    }

    // Фиксация транзакции
    return tx.Commit(ctx)
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

func (r *PostRepository) GetPostLikerIds(ctx context.Context, postId uuid.UUID, limit, offset int) ([]uuid.UUID, error) {
    query := `
        SELECT room_id
        FROM post_likes
        WHERE post_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3;
    `

    rows, err := r.db.Query(ctx, query, postId, limit, offset)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    ids := make([]uuid.UUID, 0, limit)

    for rows.Next() {
        var id uuid.UUID
        if err := rows.Scan(&id); err != nil {
            return nil, r.parseError(err)
        }
        ids = append(ids, id)
    }

    if err = rows.Err(); err != nil {
        return nil, r.parseError(err)
    }

    return ids, nil
}

func (r *PostRepository) AddLike(ctx context.Context, postId, roomId uuid.UUID) error {
    query := `INSERT INTO post_likes (post_id, room_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
    _, err := r.db.Exec(ctx, query, postId, roomId)
    return r.parseError(err)
}

func (r *PostRepository) RemoveLike(ctx context.Context, postId, roomId uuid.UUID) error {
    query := `DELETE FROM post_likes WHERE post_id = $1 AND room_id = $2`
    _, err := r.db.Exec(ctx, query, postId, roomId)
    return r.parseError(err)
}

func (r *PostRepository) CheckLikeStatus(ctx context.Context, postId, roomId uuid.UUID) (bool, error) {
    var exists bool
    query := `SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND room_id = $2)`
    err := r.db.QueryRow(ctx, query, postId, roomId).Scan(&exists)
	if err != nil {
		return exists, r.parseError(err)
	}
    return exists, err
}

func (r *PostRepository) HIncrView(ctx context.Context, postId string) error {
	err := r.redisStats.HIncrBy(ctx, "post_views_buffer", postId, 1).Err()
	if err != nil {
		return r.parseError(err)
	}
	return nil
}

func (r *PostRepository) SetView(ctx context.Context, key string, count string, time time.Duration) {
	r.redisStats.Set(ctx, key, count, time)
}

func (r *PostRepository) CheckView(ctx context.Context, key string) (int64, error) {
	alreadyViewed, err := r.redisStats.Exists(ctx, key).Result()
    if err != nil {
        return -1, r.parseError(err)
    }
	return alreadyViewed, nil
}

func (r *PostRepository) GetAllViews(ctx context.Context) (map[string]string, error) {
	viewsMap, err := r.redisStats.HGetAll(ctx, "post_views_buffer").Result()
	if err != nil {
		return nil, r.parseError(err)
	}
	r.redisStats.Del(ctx, "post_views_buffer")
	return viewsMap, nil
}

func (r *PostRepository) BulkIncrementViews(ctx context.Context, views map[string]int) error {
	if len(views) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(views))
	counts := make([]int32, 0, len(views))

	for idStr, count := range views {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue 
		}
		ids = append(ids, id)
		counts = append(counts, int32(count))
	}

	query := `
		UPDATE posts AS p
		SET views_count = p.views_count + v.inc
		FROM (
			SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS inc
		) AS v
		WHERE p.id = v.id
	`

	_, err := r.db.Exec(ctx, query, ids, counts)
	return err
}

func (r *PostRepository) GetPostForShowing(ctx context.Context, postID uuid.UUID) (*responses.ShowingPost, error) {
	query := `
		SELECT 
			p.room_id, 
			c.payload, 
			p.category_slug, 
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
		WHERE p.id = $1
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
			p.category_slug, 
			p.canvas_id, 
			p.title, 
			p.views_count, 
			p.likes_count
		FROM posts p
		WHERE p.status = 'published' 
          AND p.ai_check_status = 'passed'
          AND p.canvas_id IS NOT NULL
		ORDER BY p.published_at DESC
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

// func (r *PostRepository) GetCategoryIdBySlug(ctx context.Context, slug string) (int, error) {
//     var id int
    
//     query := `SELECT id FROM categories WHERE slug = $1 LIMIT 1`
    
//     err := r.db.QueryRow(ctx, query, slug).Scan(&id)
//     if err != nil {
//         return 0, r.parseError(err)
//     }

//     return id, nil
// }

func (r *PostRepository) CreatePost(ctx context.Context, data models.CreatePostInternal) (uuid.UUID, error) {
    var postID uuid.UUID
    err := r.db.QueryRow(ctx, `
        INSERT INTO posts (
            room_id, category_slug, title, status, ai_check_status
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, data.RoomID, data.CategorySlug, data.Title, "processing", "notChecked").Scan(&postID)

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

func (r *PostRepository) UpdateStatusUploaded(ctx context.Context, postID uuid.UUID) error {
	query := `
        UPDATE posts 
        SET 
            status = 'checking', 
            updated_at = CURRENT_TIMESTAMP 
        WHERE id = $1;
    `

    result, err := r.db.Exec(ctx, query, postID)
    if err != nil {
        return r.parseError(err) 
    }

    if result.RowsAffected() == 0 {
        return fmt.Errorf("post with id %s not found", postID)
    }

    return nil
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
        return apperrors.ErrNotFound
    }

    err = tx.Commit(ctx)
	if err != nil {
		return r.parseError(err)
	}

	return err
}

func (r *PostRepository) GetPersonalPosts(ctx context.Context, roomId uuid.UUID) ([]responses.PostInfoPersonal, error) {
    query := `
        SELECT 
            p.id, 
            p.room_id, 
            p.preview_url, 
            p.category_slug, 
            p.canvas_id, 
            p.title, 
            p.views_count, 
            p.likes_count,
            p.status,
            p.ai_check_status
        FROM posts p
        WHERE p.room_id = $1 
        ORDER BY p.created_at DESC
    `

    rows, err := r.db.Query(ctx, query, roomId)
    if err != nil {
        return nil, r.parseError(err)
    }
    defer rows.Close()

    var posts []responses.PostInfoPersonal

    for rows.Next() {
        var p responses.PostInfoPersonal
        err := rows.Scan(
            &p.Id,
            &p.RoomId,
            &p.PreviewUrl,
            &p.Category,
            &p.CanvasId,
            &p.Title,
            &p.ViewsCount,
            &p.LikesCount,
            &p.Status,
            &p.StatusAi,
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

func (r *PostRepository) GetRoomPosts(ctx context.Context, roomId uuid.UUID) ([]responses.PostInfo, error) {
    query := `
        SELECT 
            p.id, 
            p.room_id, 
            p.preview_url, 
            p.category_slug, 
            p.canvas_id, 
            p.title, 
            p.views_count, 
            p.likes_count
        FROM posts p
        WHERE p.room_id = $1 AND
		p.status = 'published'
        ORDER BY p.created_at DESC
    `

    rows, err := r.db.Query(ctx, query, roomId)
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

func NewPostRepository(db *pgxpool.Pool, redis *redis.Client, redisStats *redis.Client) *PostRepository {
	return &PostRepository{db: db, redis: redis, redisStats: redisStats}
}