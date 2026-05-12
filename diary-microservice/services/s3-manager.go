package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Manager struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
	endpoint 	  string
}

func NewS3Manager(client *s3.Client, presignClient *s3.PresignClient) *S3Manager {
	return &S3Manager{
		client:        client,
		presignClient: presignClient,
		bucketName:    os.Getenv("S3_DIARY_BUCKET_NAME"),
		endpoint: os.Getenv("S3_ENDPOINT"),
	}
}

func (s *S3Manager) getExtensionFromMimeType(mime string) string {
	ext := "." + strings.Split(mime, "/")[1]
	return  ext
}

func (s *S3Manager) getObjectKey(relativeUrl string) string {
    parts := strings.SplitN(relativeUrl, "/", 2)
    
    if len(parts) > 1 {
        return parts[1]
    }
    
    return "" 
}

func (s *S3Manager) getRelativePath(fullUrl string) string {
    // Формируем базу, которую нужно удалить "https://storage.yandexcloud.net/"
    base := fmt.Sprintf("%s/", s.endpoint)
    
    return strings.TrimPrefix(fullUrl, base)
}

func (s *S3Manager) getFullUrl(relativePath string) string {
	return fmt.Sprintf("%s/%s", s.endpoint, relativePath)
}

func (s *S3Manager) getFullUrlFromKey(key string) string {
	return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucketName, key)
}

func (s *S3Manager) GenerateUploadUrls(ctx context.Context, key string, mimeType string) (string, string, error) {
	// Генерируем публичную ссылку
	// Формат для Yandex: https://storage.yandexcloud.net/bucket/key
	publicUrl := fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucketName, key)

	// Генерируем Presigned URL для PUT-запроса
	// Мы указываем ContentType, чтобы S3 проверял его при загрузке
	presignedRequest, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(mimeType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(30 * time.Minute) 
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to sign request: %w", err)
	}

	return publicUrl, presignedRequest.URL, nil
}
