package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type S3Manager struct {
	s3Client *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

func NewS3Manager(client *s3.Client, bucket string) *S3Manager {
	return &S3Manager{
		s3Client:  client,
		presigner: s3.NewPresignClient(client),
		bucket:    bucket,
	}
}

// GetShortStaticPath формирует путь вида bucketName/roomId/fileId.ext
func (s *S3Manager) GetShortStaticPath(roomId, fileId, ext string) string {
	return fmt.Sprintf("%s/%s/%s.%s", s.bucket, roomId, fileId, ext)
}

// GetPresignedUploadURL создает подписанную ссылку для загрузки (PUT)
// Возвращает полную ссылку для фронтенда и короткий путь для сохранения в БД
func (s *S3Manager) GetPresignedUploadURL(ctx context.Context, roomId, fileId, ext string, expires time.Duration) (string, string, error) {
	// Ключ внутри бакета (без имени бакета в начале)
	key := fmt.Sprintf("%s/%s.%s", roomId, fileId, ext)
	
	request, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to sign request: %w", err)
	}

	shortPath := s.GetShortStaticPath(roomId, fileId, ext)
	return request.URL, shortPath, nil
}

// FormatFullURL превращает короткий путь из БД в полный публичный URL
func (s *S3Manager) FormatFullURL(shortPath string) string {
	if shortPath == "" {
		return ""
	}
	// Yandex Object Storage URL формат: https://storage.yandexcloud.net/bucket/key
	return fmt.Sprintf("https://storage.yandexcloud.net/%s", shortPath)
}


func (s *S3Manager) GenerateUploadContext(ctx context.Context, roomId, filename string) (string, string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".jpg" 
	}

	fileId := uuid.New().String()


	objectKey := fmt.Sprintf("%s/%s%s", roomId, fileId, ext)

	presignedReq, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	staticPath := fmt.Sprintf("%s/%s", s.bucket, objectKey)

	return presignedReq.URL, staticPath, nil
}