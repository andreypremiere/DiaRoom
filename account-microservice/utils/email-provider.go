package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/smtp"
)

// EmailConfig хранит настройки SMTP сервера
type EmailConfig struct {
	Host     string
	Port     string
	Email    string
	Password string 
}

type MailService struct {
	config EmailConfig
}

func NewMailService(cfg EmailConfig) *MailService {
	return &MailService{config: cfg}
}

func encodeSubject(subject string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(subject))
	return "=?UTF-8?B?" + encoded + "?="
}

// Отправка кода подтверждения
func (s *MailService) SendVerificationCode(toEmail, code string) error {
	// HTML-шаблон для корректного отображения в почтовиках
	const htmlTemplate = `
	<!DOCTYPE html>
	<html>
	<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
		<div style="max-width: 400px; margin: 0 auto; background-color: #ffffff; padding: 30px; border-radius: 10px; box-shadow: 0 4px 10px rgba(0,0,0,0.1);">
			<h1 style="color: #333; text-align: center; margin-top: 0;">Diaroom</h1>
			<p style="color: #666; font-size: 16px; text-align: center;">Код подтверждения:</p>
			<div style="background-color: #f3f3f3; padding: 15px; border-radius: 8px; text-align: center; font-size: 24px; font-weight: bold; letter-spacing: 5px; color: #000; margin: 20px 0;">
				{{.Code}}
			</div>
			<p style="color: #999; font-size: 12px; text-align: center; line-height: 1.5;">
				Если вы не регистрировались в системе, просто проигнорируйте это письмо.
			</p>
		</div>
	</body>
	</html>
	`

	// 2. Парсим и заполняем шаблон
	t, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := t.Execute(&body, struct{ Code string }{Code: code}); err != nil {
		return err
	}

	subject := encodeSubject("Код подтверждения")
	fromName := "Diaroom"
    
    // 2. Собираем заголовки БЕЗ лишних пробелов в начале строк
    // Важно: \r\n должен быть в конце каждой строки заголовка
    header := ""
    header += fmt.Sprintf("From: %s <%s>\r\n", fromName, s.config.Email)
    header += fmt.Sprintf("To: %s\r\n", toEmail)
    header += fmt.Sprintf("Subject: %s\r\n", subject) // Тема здесь!
    header += "MIME-Version: 1.0\r\n"
    header += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
    header += "\r\n" // Пустая строка отделяет заголовки от тела письма

    // 3. Соединяем заголовки и тело
    message := append([]byte(header), body.Bytes()...)

    // Настройки Яндекса
    auth := smtp.PlainAuth("", s.config.Email, s.config.Password, s.config.Host)
    
    return smtp.SendMail(s.config.Host + ":" + s.config.Port, auth, s.config.Email, []string{toEmail}, message)
}