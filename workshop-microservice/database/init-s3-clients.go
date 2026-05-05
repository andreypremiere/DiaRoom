package database

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewS3Client() (*s3.Client, *s3.PresignClient, error) {
	accessKey := os.Getenv("S3_ACCESS_KEY_WORKSHOP_MANAGER")
	secretKey := os.Getenv("S3_SECRET_KEY_WORKSHOP_MANAGER")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ru-central1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey, 
			secretKey, 
			"",
		)),
	)
	if err != nil {
		return nil, nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://storage.yandexcloud.net")
	})

	presignedClient := s3.NewPresignClient(client)

	return client, presignedClient, nil
}