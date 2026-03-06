package repositories

import (
	"context"
	"room-microservice/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepositoryInter interface {
	NewRoom(
	ctx context.Context, id uuid.UUID, 
	userId uuid.UUID, roomName string, roomNameId string) error
	GetRoomIdByUserId(ctx context.Context, userId uuid.UUID) (uuid.UUID, error)
	GetRoomByRoomId(ctx context.Context, roomId uuid.UUID) (*models.RoomExpanded, error)
}

type roomResository struct {
	db *pgxpool.Pool
}

func (rp *roomResository) NewRoom(
	ctx context.Context, id uuid.UUID, 
	userId uuid.UUID, 
	roomName string, 
	roomNameId string) error {
		sqlString := "INSERT INTO rooms (id, user_id, room_name, room_name_id) VALUES($1, $2, $3, $4)"
		_, err := rp.db.Exec(ctx, sqlString, id, userId, roomName, roomNameId)
		return err
}

func (rp *roomResository) GetRoomIdByUserId(ctx context.Context, userId uuid.UUID) (uuid.UUID, error) {
	var roomId uuid.UUID
	sqlString := "SELECT id FROM rooms WHERE user_id = $1"
	err := rp.db.QueryRow(ctx, sqlString, userId).Scan(&roomId)
	if err != nil {
		return uuid.Nil, err
	}
	return roomId, nil
}

func (rp *roomResository) GetRoomByRoomId(ctx context.Context, roomId uuid.UUID) (*models.RoomExpanded, error) {
	room := models.RoomExpanded{}
	sqlString := `
    SELECT 
        r.id, r.user_id, r.room_name, r.room_name_id, r.avatar_url, r.bio, r.settings, 
        r.followers_count, r.following_count,
        COALESCE(
            (SELECT json_agg(json_build_object('slug', c.slug, 'name', c.name))
             FROM categories c
             JOIN room_categories rc ON c.id = rc.category_id
             WHERE rc.room_id = r.id), 
            '[]'
        ) as categories
    FROM rooms r
    WHERE r.id = $1
	`

	err := rp.db.QueryRow(ctx, sqlString, roomId).Scan(
		&room.Id,            
		&room.UserId,        
		&room.RoomName,      
		&room.RoomNameId,    
		&room.AvatarUrl,     
		&room.Bio,          
		&room.Settings,      
		&room.FollowersCount, 
		&room.FollowingCount,
		&room.Categories,
	)
	return &room, err
}

func NewRoomRepository(db *pgxpool.Pool) *roomResository {
	return &roomResository{db}
}