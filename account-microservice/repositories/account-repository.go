package repositories

import (
	apperrors "account-microservice/app-errors"
	"account-microservice/contracts/account/requests"
	"account-microservice/contracts/account/responses"
	"account-microservice/models"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AccountRepository struct {
	poolPg      *pgxpool.Pool
	redisClient *redis.Client
}

func (r *AccountRepository) CheckSubscription(ctx context.Context, followerId, followingId uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 
			FROM subscriptions 
			WHERE follower_id = $1 AND following_id = $2
		);`

	err := r.poolPg.QueryRow(ctx, query, followerId, followingId).Scan(&exists)
	if err != nil {
		return false, r.parseError(err)
	}
	return exists, nil
}

func (r *AccountRepository) AddSubscription(ctx context.Context, followerId, followingId uuid.UUID) error {
	query := `
		INSERT INTO subscriptions (follower_id, following_id) 
		VALUES ($1, $2) 
		ON CONFLICT (follower_id, following_id) DO NOTHING;`
	_, err := r.poolPg.Exec(ctx, query, followerId, followingId)
	if err != nil {
		return r.parseError(err)
	}
	return nil
}

func (r *AccountRepository) RemoveSubscription(ctx context.Context, followerId, followingId uuid.UUID) error {
	query := `
		DELETE FROM subscriptions 
		WHERE follower_id = $1 AND following_id = $2;`

	_, err := r.poolPg.Exec(ctx, query, followerId, followingId)
	if err != nil {
		return r.parseError(err)
	}
	return nil
}

func (ar AccountRepository) GetRoomInfo(context context.Context, id uuid.UUID) (*responses.RoomInfo, error) {
	query := `
		SELECT avatar_url, room_name 
		FROM rooms 
		WHERE id = $1
		LIMIT 1
	`

	var info responses.RoomInfo

	err := ar.poolPg.QueryRow(context, query, id).Scan(
		&info.AvatarUrl,
		&info.RoomName,
	)

	if err != nil {
		return nil, ar.parseError(err)
	}

	return &info, nil
}

func (ar AccountRepository) UpdateRoom(ctx context.Context, roomId uuid.UUID, request *requests.UpdateRoomRequest) error {
	tx, err := ar.poolPg.Begin(ctx)
	if err != nil {
		return ar.parseError(err)
	}
	defer tx.Rollback(ctx)

	roomQuery := `
        UPDATE rooms 
        SET 
            room_name = $1, 
            room_unique_id = $2, 
            bio = $3, 
            avatar_url = $4, 
            background_url = $5
        WHERE id = $6
        RETURNING user_id` 

	var userId uuid.UUID
	err = tx.QueryRow(ctx, roomQuery,
		request.RoomName,
		request.RoomUniqueID,
		request.Bio,
		request.AvatarFileName,
		request.BackgroundFileName,
		roomId,
	).Scan(&userId)

	if err != nil {
		return ar.parseError(err)
	}

	userQuery := `UPDATE users SET is_configured = true WHERE id = $1`
	_, err = tx.Exec(ctx, userQuery, userId)
	if err != nil {
		return ar.parseError(err)
	}

	_, err = tx.Exec(ctx, "DELETE FROM room_categories WHERE room_id = $1", roomId)
	if err != nil {
		return ar.parseError(err)
	}

	if len(request.Categories) > 0 {
		for _, slug := range request.Categories {
			_, err = tx.Exec(ctx, `
                INSERT INTO room_categories (room_id, category_slug) 
                VALUES ($1, $2)`,
				roomId, slug,
			)
			if err != nil {
				return ar.parseError(err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ar.parseError(err)
	}

	return nil
}

func (ar AccountRepository) GetRoom(context context.Context, roomId uuid.UUID) (*responses.RoomResponse, error) {
	var room responses.RoomResponse

	query := `
        SELECT 
            room_unique_id, 
            room_name, 
            COALESCE(bio, ''), 
            COALESCE(avatar_url, ''), 
            COALESCE(background_url, ''),
            COALESCE(
                (SELECT array_agg(category_slug) 
                 FROM room_categories 
                 WHERE room_id = rooms.id), 
                '{}'
            )
        FROM rooms 
        WHERE id = $1`

	err := ar.poolPg.QueryRow(context, query, roomId).Scan(
		&room.RoomUniqueId,
		&room.RoomName,
		&room.Bio,
		&room.AvatarPath,
		&room.BackgroundPath,
		&room.ListCategory,
	)

	if err != nil {
		return nil, ar.parseError(err)
	}

	return &room, nil
}

func (ar *AccountRepository) AddCodeWithTimeout(
	ctx context.Context, userId uuid.UUID, code string) error {
	err := ar.redisClient.Set(ctx, userId.String(), code, 2*time.Minute).Err()
	if err != nil {
		return ar.parseError(err)
	}
	return nil
}

// func (ar *AccountRepository) GetRoomInfoById(ctx context.Context, id uuid.UUID) (responses.RoomInfo, error) {
// 	query := `
//         SELECT room_name, avatar_url 
//         FROM rooms 
//         WHERE id = $1
//     `

//     var info responses.RoomInfo
//     err := ar.poolPg.QueryRow(ctx, query, id).Scan(
//         &info.RoomName,
//         &info.AvatarUrl,
//     )

//     if err != nil {
//         return responses.RoomInfo{}, ar.parseError(err)
//     }

//     return info, nil
// }

func (ar *AccountRepository) GetRoomsInfoByIds(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]responses.RoomInfo, error) {
	query := `
        SELECT id, room_name, avatar_url 
        FROM rooms 
        WHERE id = ANY($1)
    `

	rows, err := ar.poolPg.Query(ctx, query, ids)
	if err != nil {
		return nil, ar.parseError(err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]responses.RoomInfo)

	for rows.Next() {
		var id uuid.UUID
		var info responses.RoomInfo

		err := rows.Scan(
			&id,
			&info.RoomName,
			&info.AvatarUrl,
		)
		if err != nil {
			return nil, ar.parseError(err)
		}

		result[id] = info
	}

	if err = rows.Err(); err != nil {
		return nil, ar.parseError(err)
	}

	return result, nil
}

func (ar *AccountRepository) NewAccount(ctx context.Context, email string, userID, roomID uuid.UUID, roomUniqueId, roomName, hashPassword string) error {
	tx, err := ar.poolPg.Begin(ctx)
	if err != nil {
		return ar.parseError(err)
	}

	defer tx.Rollback(ctx)

	userQuery := `
		INSERT INTO users (id, email, hash_password) 
		VALUES ($1, $2, $3)`

	_, err = tx.Exec(ctx, userQuery, userID, email, hashPassword)
	if err != nil {
		return ar.parseError(err)
	}

	roomQuery := `
		INSERT INTO rooms (id, user_id, room_unique_id, room_name) 
		VALUES ($1, $2, $3, $4)`

	_, err = tx.Exec(ctx, roomQuery, roomID, userID, roomUniqueId, roomName)
	if err != nil {
		return ar.parseError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ar.parseError(err)
	}

	return nil
}

func (ar *AccountRepository) parseError(err error) error {
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

func (ar *AccountRepository) GetOTPCode(ctx context.Context, userId uuid.UUID) (string, error) {
	code, err := ar.redisClient.Get(ctx, userId.String()).Result()

	if err != nil {
		return "", ar.parseError(err)
	}

	return code, nil
}

func (ar *AccountRepository) GetStatusConfigured(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isConfigured bool
	query := `SELECT is_configured FROM users WHERE id = $1`

	err := ar.poolPg.QueryRow(ctx, query, userID).Scan(&isConfigured)
	if err != nil {
		return false, ar.parseError(err)
	}

	return isConfigured, nil
}

func (ar *AccountRepository) GetRoomIdByUserId(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var roomID uuid.UUID
	query := `SELECT id FROM rooms WHERE user_id = $1`

	err := ar.poolPg.QueryRow(ctx, query, userID).Scan(&roomID)
	if err != nil {
		return uuid.Nil, ar.parseError(err)
	}

	return roomID, nil
}

func (ar *AccountRepository) UpdateRefreshToken(
	ctx context.Context,
	oldToken string,
	newToken string,
	newExpiresAt time.Time,
) error {
	query := `
        UPDATE sessions 
        SET refresh_token = $1, expires_at = $2 
        WHERE refresh_token = $3
    `

	result, err := ar.poolPg.Exec(ctx, query, newToken, newExpiresAt, oldToken)
	if err != nil {
		return ar.parseError(err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

func (ar *AccountRepository) GetUserByEmail(ctx context.Context, email string) (*models.BaseUser, error) {
	user := &models.BaseUser{}

	query := `SELECT id, email, hash_password, is_activated FROM users WHERE email = $1`

	err := ar.poolPg.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.IsActivated,
	)

	if err != nil {
		return nil, ar.parseError(err)
	}

	return user, nil
}

func (ar *AccountRepository) VerifyAndCreateSession(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken string,
	deviceInfo string,
	expiresAt time.Time,
) error {
	tx, err := ar.poolPg.Begin(ctx)
	if err != nil {
		return ar.parseError(err)
	}

	defer tx.Rollback(ctx)

	const updateStatusQuery = `UPDATE users SET is_activated = true WHERE id = $1`
	_, err = tx.Exec(ctx, updateStatusQuery, userID)
	if err != nil {
		return ar.parseError(err)
	}

	const addTokenQuery = `
        INSERT INTO sessions (user_id, refresh_token, user_agent, expires_at)
        VALUES ($1, $2, $3, $4)
    `
	_, err = tx.Exec(ctx, addTokenQuery, userID, refreshToken, deviceInfo, expiresAt)
	if err != nil {
		return ar.parseError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ar.parseError(err)
	}

	return nil
}

func (ar *AccountRepository) GetSessionByToken(ctx context.Context, token string) (*models.SessionWithRoomId, error) {
	session := &models.SessionWithRoomId{}

	query := `
        SELECT 
            rt.user_id, 
            rt.refresh_token, 
            rt.user_agent, 
            rt.expires_at,
            r.id as room_id
        FROM sessions rt
        JOIN rooms r ON rt.user_id = r.user_id
        WHERE rt.refresh_token = $1
    `

	err := ar.poolPg.QueryRow(ctx, query, token).Scan(
		&session.UserId,
		&session.RefreshToken,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.RoomId,
	)

	if err != nil {
		return nil, ar.parseError(err)
	}

	return session, nil
}

func (ar *AccountRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE refresh_token = $1`

	_, err := ar.poolPg.Exec(ctx, query, token)
	if err != nil {
		return ar.parseError(err)
	}

	return nil
}

func (ar *AccountRepository) GetUserEmailByID(ctx context.Context, userID uuid.UUID) (*models.EmailUser, error) {
	var user models.EmailUser

	query := `SELECT id, email FROM users WHERE id = $1`
	err := ar.poolPg.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Email)

	if err != nil {
		return nil, ar.parseError(err)
	}
	return &user, nil
}

func NewAccountRepository(
	poolPg *pgxpool.Pool,
	redisClient *redis.Client,
) *AccountRepository {
	return &AccountRepository{
		poolPg:      poolPg,
		redisClient: redisClient,
	}
}
