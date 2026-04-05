package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher предоставляет методы для безопасной работы с паролями
type PasswordHasher struct {
	cost int 
}

func NewPasswordHasher(cost int) *PasswordHasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &PasswordHasher{cost: cost}
}

// HashPassword превращает обычную строку в безопасный bcrypt-хэш
func (ph *PasswordHasher) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), ph.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ComparePassword проверяет, соответствует ли введенный пароль сохраненному хэшу
func (ph *PasswordHasher) ComparePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}