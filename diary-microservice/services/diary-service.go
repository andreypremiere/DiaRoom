package services

import (
	"diary-microservice/repositories"

	"github.com/redis/go-redis/v9"
)

type DiaryService struct {
	repo  *repositories.DiaryRepository
	s3    *S3Manager
	redis *redis.Client
}

func NewDiaryService(repo *repositories.DiaryRepository, s3 *S3Manager, redisClient *redis.Client) *DiaryService {
	return &DiaryService{
		repo:  repo,
		s3:    s3,
		redis: redisClient,
	}
}