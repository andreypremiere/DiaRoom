package models

import (
	"diary-microservice/contracts/requests"

	"github.com/google/uuid"
)

type Tag struct {
	Id uuid.UUID `json:"id"`
	RoomId uuid.UUID `json:"roomId"`
	Name string `json:"name"`
	Color int64 `json:"color"`
}

func FromCreatingTag(tag *requests.CreatingTag, roomId uuid.UUID, tagId uuid.UUID) *Tag {
	return &Tag{
		Id: tagId,
		RoomId: roomId,
		Name: tag.Name,
		Color: tag.Color,
	}
}