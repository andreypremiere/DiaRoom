package services

import (
	"os"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Manager struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

func NewS3Manager(client *s3.Client, presignClient *s3.PresignClient) *S3Manager {
	return &S3Manager{
		client:        client,
		presignClient: presignClient,
		bucketName:    os.Getenv("S3_DIARY_BUCKET_NAME"),
	}
}

// Здесь методы S3Manager