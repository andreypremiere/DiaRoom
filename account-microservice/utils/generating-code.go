package utils

import (
	"crypto/rand"
	"math/big"
	"strconv"
)

// GenerateCode генерирует случайный шестизначный код подтверждения (OTP)
//
// :return: Строка с кодом от 100000 до 999999 и ошибка генерации
func GenerateCode() (string, error) {
	// Определение диапазона для генерации (900,000 вариантов)
	max := big.NewInt(900000)
	
	// Использование криптографически стойкого генератора случайных чисел
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Смещение результата, чтобы код всегда был шестизначным
	code := 100000 + n.Int64()
	
	// Преобразование числового кода в строковый формат
	return strconv.Itoa(int(code)), nil
}