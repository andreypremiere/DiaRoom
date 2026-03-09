package services

import (
	"context"
	"errors"
	"room-microservice/models"
	"room-microservice/repositories"

	"github.com/google/uuid"
)

// RoomServiceInter описывает бизнес-логику для работы с комнатами
type RoomServiceInter interface {
	// AddRoom проверяет входные данные, генерирует UUID и создает новую комнату
	AddRoom(ctx context.Context, room *models.BaseRoom) (uuid.UUID, error) 

	// GetRoomIdByUserId возвращает идентификатор комнаты, привязанной к пользователю
	GetRoomIdByUserId(ctx context.Context, userId uuid.UUID) (uuid.UUID, error)

	// GetRoomByRoomId возвращает полную информацию о комнате по ее идентификатору
	GetRoomByRoomId(ctx context.Context, roomId uuid.UUID) (*models.RoomExpanded, error)
}

// roomService реализует интерфейс бизнес-логики и взаимодействует с репозиторием
type roomService struct {
	roomRepo repositories.RoomRepositoryInter
}

// AddRoom проверяет входные данные и создает новую комнату с генерацией UUID
func (rs *roomService) AddRoom(ctx context.Context, room *models.BaseRoom) (uuid.UUID, error) {
	// Генерация нового идентификатора для комнаты
	newRoomId := uuid.New()

	// Валидация данных
	if room.RoomName == "" {
		return newRoomId, errors.New("RoomName cannot be empty")
	}

	if room.RoomNameId == "" {
		return newRoomId, errors.New("RoomNameId cannot be empty")
	}

	// Вызов метода репозитория для сохранения в БД
	err := rs.roomRepo.NewRoom(ctx, newRoomId, room.UserId, room.RoomName, room.RoomNameId)
	return newRoomId, err
	
}

// GetRoomIdByUserId получает ID комнаты через репозиторий по ID пользователя
func (rs *roomService) GetRoomIdByUserId(ctx context.Context, userId uuid.UUID) (uuid.UUID, error) {
	roomId, err := rs.roomRepo.GetRoomIdByUserId(ctx, userId)
	return roomId, err
}

// GetRoomByRoomId возвращает полную информацию о комнате с обработкой ошибок
func (rs *roomService) GetRoomByRoomId(ctx context.Context, roomId uuid.UUID) (*models.RoomExpanded, error) {
	room, err := rs.roomRepo.GetRoomByRoomId(ctx, roomId)
	if err != nil {
		return nil, errors.Join(errors.New("Ошибка на этапе получения комнаты из базы данных"), err)
	}
	return room, nil 
}

// NewRoomService создает новый экземпляр сервиса с внедрением зависимости репозитория
func NewRoomService(roomRepo repositories.RoomRepositoryInter) *roomService {
	return &roomService{roomRepo}
}