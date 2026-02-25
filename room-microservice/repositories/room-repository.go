package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepositoryInter interface {
	NewRoom(
	ctx context.Context, id uuid.UUID, 
	userId uuid.UUID, roomName string, roomNameId string) error
}

type roomResository struct {
	db *pgxpool.Pool
}

func (rp *roomResository) NewRoom(
	ctx context.Context, id uuid.UUID, 
	userId uuid.UUID, roomName string, roomNameId string) error {
		sqlString := "INSERT INTO rooms (id, user_id, room_name, room_name_id) VALUES($1, $2, $3, $4)"
		_, err := rp.db.Exec(ctx, sqlString, id, userId, roomName, roomNameId)
		return err
	}

func NewRoomRepository(db *pgxpool.Pool) *roomResository {
	return &roomResository{db}
}