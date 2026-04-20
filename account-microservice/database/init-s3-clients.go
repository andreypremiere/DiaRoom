package database

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func InitS3Client() *s3.Client {
	// 1. Создаем провайдер учетных данных
	appCreds := credentials.NewStaticCredentialsProvider(
		os.Getenv("S3_ACCESS_KEY_AVATARS_MANAGER"),
		os.Getenv("S3_SECRET_KEY_AVATARS_MANAGER"),
		"",
	)

	// 2. Загружаем базовый конфиг
	// Указываем регион Яндекса ru-central1
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ru-central1"),
		config.WithCredentialsProvider(appCreds),
	)
	if err != nil {
		log.Fatal("Ошибка загрузки конфига S3:", err)
	}

	// 3. Создаем клиент S3, явно указывая Endpoint Яндекса
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://storage.yandexcloud.net")
        // Яндекс требует использования Path-Style ссылок в некоторых случаях
        o.UsePathStyle = true 
	})

	return client
}