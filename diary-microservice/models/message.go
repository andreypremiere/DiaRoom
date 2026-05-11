package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID                       uuid.UUID  `json:"id" db:"id"`
	RoomID                   uuid.UUID  `json:"roomId" db:"room_id"`
	MsgType                  string     `json:"msgType" db:"msg_type"` // standard, voice_note, video_note
	Content                  *string    `json:"content" db:"content"`   
	Status 					 *string    `json:"status" db:"status"`
	AttachedObjectWorkshopID *uuid.UUID `json:"attachedObjectWorkshopId" db:"attached_object_workshop_id"`
	AttachedObjectPostID     *uuid.UUID `json:"attachedObjectPostId" db:"attached_object_post_id"`
	CreatedAt                time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt                time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt                *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}