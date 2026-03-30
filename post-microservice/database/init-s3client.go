package database

import (
	"context"
	// "fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/joho/godotenv"
)

func NewS3Client() (*s3.Client, error) {
	// if err := godotenv.Load(); err != nil {
	// 	// Не всегда ошибка критична (например, в Docker переменные прокинуты напрямую)
	// 	fmt.Println("Предупреждение: .env файл не найден")
	// }

	// 2. Достаем переменные из окружения
	accessKey := os.Getenv("S3_ACCESS_KEY_MEDIA_MANAGER")
	secretKey := os.Getenv("S3_SECRET_KEY_MEDIA_MANAGER")

	// 1. Загружаем дефолтный конфиг
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		// Указываем регион Яндекса
		config.WithRegion("ru-central1"),
		// Передаем твои статические ключи доступа
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey, 
			secretKey, 
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	// 2. Создаем клиент, указывая Endpoint напрямую в его настройках
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://storage.yandexcloud.net")
	})

	return client, nil
}