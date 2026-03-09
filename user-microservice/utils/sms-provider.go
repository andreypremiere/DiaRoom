package utils

import (
	"fmt"
)

// SmsProvider определяет интерфейс для отправки кодов подтверждения
type SmsProvider interface {
	// SendCode отправляет шестизначный код на указанный номер телефона
	SendCode(phone string, code string) error
}

// ConsoleSms реализует интерфейс SmsProvider для локальной разработки и отладки
type ConsoleSms struct {
}

// SendCode выводит проверочный код в стандартный поток вывода (консоль)
//
// :param phone: Номер телефона получателя
// :param code: Сгенерированный код подтверждения
// :return: Всегда возвращает nil в данной реализации
func (cs ConsoleSms) SendCode(phone string, code string) error {
	// Эмуляция отправки сообщения через вывод в терминал
	fmt.Println("Высланный код:", code, "для пользователя:", phone)
	return nil
}