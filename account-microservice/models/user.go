package models

import "github.com/google/uuid"

type BaseUser struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	IsActivated  bool
}
