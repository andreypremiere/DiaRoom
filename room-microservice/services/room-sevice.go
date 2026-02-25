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
}

type roomService struct {
	userRepo repositories.RoomRepositoryInter
}

func (rs *roomService) AddRoom(ctx context.Context, room *models.BaseRoom) (uuid.UUID, error) {
	newRoomId := uuid.New()

	if room.RoomName == "" {
		return newRoomId, errors.New("RoomName cannot be empty")
	}

	if room.RoomNameId == "" {
		return newRoomId, errors.New("RoomNameId cannot be empty")
	}

	err := rs.userRepo.NewRoom(ctx, newRoomId, room.UserId, room.RoomName, room.RoomNameId)
	return newRoomId, err
	
}


func NewRoomService(roomRepo repositories.RoomRepositoryInter) *roomService {
	return &roomService{roomRepo}
}