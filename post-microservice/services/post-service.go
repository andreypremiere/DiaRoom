package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	// "path/filepath"
	"post-microservice/clients"
	"post-microservice/contracts/responses"
	"post-microservice/models"
	"post-microservice/repositories"

	// "strings"
	// "time"

	// "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	// "github.com/google/uuid"
)

type PostServiceInter interface {
    CreatePost(ctx context.Context, req models.CreatePostRequest) (*models.CreatePostResponse, error)
	GenerateMediaUrls(ctx context.Context, roomID uuid.UUID, req models.GenerateUrlsRequest) (*models.GenerateUrlsResponse, error)
	CreateAndAttachCanvas(ctx context.Context, postID uuid.UUID, payload json.RawMessage) error
	GetAllPosts(ctx context.Context) ([]responses.Post, error)
}

type PostService struct {
	repo repositories.PostRepositoryInter
	s3Client   *s3.Client
	bucketMediaName string
	accountClient *clients.AccountClient
}

func (s *PostService) GetAllPosts(ctx context.Context) ([]responses.Post, error) {
	postsInfo, err := s.repo.GetAllPosts(ctx)
	if err != nil {
		return nil, errors.New("Ошибка при запросе постов в бд")
	}

	for i := range postsInfo {
		postsInfo[i].PreviewUrl = fmt.Sprintf("https://storage.yandexcloud.net/%s", postsInfo[i].PreviewUrl)
	}

	if len(postsInfo) == 0 {
		return []responses.Post{}, nil
	}

	roomIDs := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]bool)
	for _, p := range postsInfo {
		if !seen[p.RoomId] {
			seen[p.RoomId] = true
			roomIDs = append(roomIDs, p.RoomId)
		}
	}

	roomsInfo, err := s.accountClient.GetAuthorsBatch(ctx, roomIDs)
	if err != nil {
		fmt.Println("Ошибка получения roomsInfo в post-service" + err.Error())
	}

	result := make([]responses.Post, len(postsInfo))
	for i, info := range postsInfo {
		result[i] = responses.Post{
			PostInfo: info,
			RoomInfo: roomsInfo[info.RoomId], 
		}
	}

	return result, nil
}

func (s *PostService) CreateAndAttachCanvas(ctx context.Context, postID uuid.UUID, payload json.RawMessage) error {
	err := s.repo.InsertCanvasAndUpdatePost(ctx, postID, payload)
	if (err != nil) {
		fmt.Println("Ошибка добавления холста в пост", err.Error())
		return err
	}

	err = s.repo.PushPostToQueue(ctx, postID)
	if (err != nil) {
		fmt.Println("Ошибка добавления id поста в очередь", err.Error())
		return err
	}
	// Делегируем работу с БД репозиторию
	return nil
}

func (s *PostService) CreatePost(ctx context.Context, req models.CreatePostRequest) (*models.CreatePostResponse, error) {
    // 1. Находим category_id по slug
	categoryID, err := s.repo.GetCategoryIdBySlug(ctx, req.Post.CategorySlug)
	if err != nil {
        return nil, fmt.Errorf("категория не найдена: %s", req.Post.CategorySlug)
	}

    fmt.Println("Категория была найдена: ", categoryID)

    // 2. Создаём пост и получаем его ID
	postID, err := s.repo.CreatePost(ctx, models.CreatePostInternal{
		RoomID:     req.Post.RoomID,
		CategoryID: categoryID,
		Title:      req.Post.Title,
		Status:     req.Post.PostStatus,
		AiStatus:   req.Post.AiStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пост: %w", err)
	}

    fmt.Println("Пост был создан", postID)

    fileExtension := path.Ext(req.Preview.Filename)
    if fileExtension == "" {
		fileExtension = ".jpg" // fallback
	}

    // 3. Формируем пути для preview
	objectKey := fmt.Sprintf("%s/%s/%s%s", req.Post.RoomID, postID, req.Preview.UploadID, fileExtension)

	// 4. Генерируем Presigned URL и Public URL для превью
	presignedURL, publicURL, err := s.generatePresignedAndPublicURL(ctx, objectKey, req.Preview.ContentType)
	if err != nil {
		return nil, fmt.Errorf("не удалось сгенерировать ссылки для preview: %w", err)
	}

    // 5. Обновляем превью для поста
    err = s.repo.UpdatePostPreviewURL(ctx, postID, publicURL)
    if err != nil {
        return nil, fmt.Errorf("не удалось обновить preview_url: %w", err)
    }

    // 6.  Добавляем хештеги
    if len(req.Post.Hashtags) > 0 {
        cleanHashtags := CleanHashtags(req.Post.Hashtags)

		if err := s.repo.AddHashtagsToPost(ctx, postID, cleanHashtags); err != nil {
			// Не прерываем создание поста, если хэштеги не сохранились
			fmt.Printf("Warning: failed to add hashtags to post %s: %v", postID, err)
		}
	}

	// 6. Формируем ответ
	return &models.CreatePostResponse{
		PostID: postID,
		Preview: models.PreviewLinksResponse{
			PublicURL:    publicURL,
			PresignedURL: presignedURL,
		},
	}, nil

}

func (s *PostService) GenerateMediaUrls(ctx context.Context, roomID uuid.UUID, req models.GenerateUrlsRequest) (*models.GenerateUrlsResponse, error) {
	// Базовая валидация (защита от пустых запросов)
	if len(req.Files) == 0 {
		return &models.GenerateUrlsResponse{
            Files: []models.GeneratedURL{}, // Инициализируем пустой слайс, чтобы Flutter не получил null
        }, nil
	}

	response := &models.GenerateUrlsResponse{
		Files: make([]models.GeneratedURL, 0, len(req.Files)), // Аллоцируем память сразу
	}

	for _, file := range req.Files {
		// 1. Извлекаем расширение файла
		ext := path.Ext(file.Filename)
		if ext == "" {
			ext = ".jpg" // Fallback по умолчанию
		}

		// 2. Формируем безопасный objectKey
		// Структура: room_id/post_id/upload_id.ext
		objectKey := fmt.Sprintf("%s/%s/%s%s",
			roomID.String(),
			req.PostID.String(),
			file.UploadID.String(),
			ext,
		)

		// 3. Вызываем твой метод генерации ссылок (предполагается, что он реализован ниже)
		presignedURL, publicURL, err := s.generatePresignedAndPublicURL(ctx, objectKey, file.ContentType)
		if err != nil {
			// Логируем, но не роняем весь процесс из-за одного битого файла
			// В реальном проекте тут лучше использовать логгер: log.Printf("ошибка генерации: %v", err)
			return nil, fmt.Errorf("ошибка генерации ссылки для файла %s: %w", file.UploadID, err)
		}

		// 4. Добавляем результат в ответ
		response.Files = append(response.Files, models.GeneratedURL{
			UploadID:     file.UploadID,
			PublicURL:    publicURL,
			PresignedURL: presignedURL,
		})
	}

	return response, nil
}

func CleanHashtags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	uniqueMap := make(map[string]struct{})
	for _, t := range tags {
		clean := strings.ToLower(strings.TrimSpace(t))
		// Убираем пустые строки и, например, одиночные знаки решетки
		if clean != "" && clean != "#" {
			// Если тег пришел с решеткой "#music", убираем её для хранения в БД
			clean = strings.TrimPrefix(clean, "#")
			uniqueMap[clean] = struct{}{}
		}
	}

	result := make([]string, 0, len(uniqueMap))
	for t := range uniqueMap {
		result = append(result, t)
	}
	return result
}

// Вспомогательный метод генерации ссылок
func (s *PostService) generatePresignedAndPublicURL(ctx context.Context, objectKey string, contentType string) (presignedURL string, publicURL string, err error) {
	presigner := s3.NewPresignClient(s.s3Client)

	// Presigned URL для загрузки
	presignReq, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketMediaName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, func(o *s3.PresignOptions) {
		o.Expires = 15 * time.Minute
	})
	if err != nil {
		return "", "", err
	}

	presignedURL = presignReq.URL
	publicURL = fmt.Sprintf("%s/%s", s.bucketMediaName, objectKey)

	return presignedURL, publicURL, nil
}

// func (s *PostService) generatePresignedPutURL(ctx context.Context, objectKey, contentType string) (string, error) {
// 	presigner := s3.NewPresignClient(s.s3Client)

// 	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
// 		Bucket:      aws.String(s.bucketMediaName),
// 		Key:         aws.String(objectKey),
// 		ContentType: aws.String(contentType),
// 	}, func(opts *s3.PresignOptions) {
// 		opts.Expires = 15 * time.Minute // ссылка живёт 15 минут
// 	})

// 	if err != nil {
// 		return "", err
// 	}

// 	return req.URL, nil
// }

func NewPostService(repository repositories.PostRepositoryInter, s3Client *s3.Client, bucketMediaName string, accountClient *clients.AccountClient) *PostService {
	return &PostService{repo: repository, s3Client: s3Client, bucketMediaName: bucketMediaName, accountClient: accountClient}
}

