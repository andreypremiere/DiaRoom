package models

import "github.com/google/uuid"

type BaseUser struct {
	Id uuid.UUID `json:"id"`
	Email string `json:"email"`
	IsActivated bool `json:"isActivated"`
}

type CheckUser struct {
	BaseUser
	HashPassword string `json:"hashPassword"`
}


