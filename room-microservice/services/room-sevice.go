package services

import (
	"context"
	"errors"
	"room-microservice/models"
	"room-microservice/repositories"

	"github.com/google/uuid"
)

type RoomServiceInter interface {
	AddRoom(ctx context.Context, room *models.BaseRoom) (uuid.UUID, error) 
	GetRoomIdByUserId(ctx context.Context, userId uuid.UUID) (uuid.UUID, error)
	GetRoomByRoomId(ctx context.Context, roomId uuid.UUID) (*models.RoomExpanded, error)
}

type roomService struct {
	roomRepo repositories.RoomRepositoryInter
}

func (rs *roomService) AddRoom(ctx context.Context, room *models.BaseRoom) (uuid.UUID, error) {
	newRoomId := uuid.New()

	if room.RoomName == "" {
		return newRoomId, errors.New("RoomName cannot be empty")
	}

	if room.RoomNameId == "" {
		return newRoomId, errors.New("RoomNameId cannot be empty")
	}

	err := rs.roomRepo.NewRoom(ctx, newRoomId, room.UserId, room.RoomName, room.RoomNameId)
	return newRoomId, err
	
}

func (rs *roomService) GetRoomIdByUserId(ctx context.Context, userId uuid.UUID) (uuid.UUID, error) {
	roomId, err := rs.roomRepo.GetRoomIdByUserId(ctx, userId)
	return roomId, err
}

func (rs *roomService) GetRoomByRoomId(ctx context.Context, roomId uuid.UUID) (*models.RoomExpanded, error) {
	room, err := rs.roomRepo.GetRoomByRoomId(ctx, roomId)
	if err != nil {
		return nil, errors.Join(errors.New("Ошибка на этапе получения комнаты из базы данных"), err)
	}
	return room, nil 
}


func NewRoomService(roomRepo repositories.RoomRepositoryInter) *roomService {
	return &roomService{roomRepo}
}