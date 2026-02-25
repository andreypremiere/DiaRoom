package utils

import (
	"fmt"
)

type SmsProvider interface {
	SendCode(phone string, code string) error
}

type ConsoleSms struct {
}


func (cs ConsoleSms) SendCode(phone string, code string) error {
	fmt.Println("Высланный код:", code, "для пользователя:", phone)
	return nil
}
