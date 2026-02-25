package utils

import (
	"crypto/rand"
	"math/big"
	"strconv"
)

func GenerateCode() (string, error) {
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	code := 100000 + n.Int64()
	return strconv.Itoa(int(code)), nil
}