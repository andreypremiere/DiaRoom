package requests

import "github.com/google/uuid"

type MoveItem struct {
	TargetId      uuid.UUID  `json:"targetId"`
	DestinationId *uuid.UUID `json:"destinationId"`
}