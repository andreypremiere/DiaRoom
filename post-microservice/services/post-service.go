package services

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	// "path/filepath"
	"post-microservice/models"
	"post-microservice/repositories"

	// "strings"
	// "time"

	// "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/google/uuid"
)

type PostServiceInter interface {
	// GetPresignedUrls(ctx context.Context, req *models.PresignedRequest, roomID string) (*models.PresignedResponse, error)
    CreatePost(ctx context.Context, req models.CreatePostRequest) (*models.CreatePostResponse, error)
    // PublishPost(ctx context.Context, req models.PublishPostRequest) error
}

type PostService struct {
	repo repositories.PostRepositoryInter
	s3Client   *s3.Client
	bucketMediaName string
}

// func (s *PostService) GetPresignedUrls(ctx context.Context, req *models.PresignedRequest, roomID string) (*models.PresignedResponse, error) {
// 	resp := &models.PresignedResponse{
// 		Files: make([]models.PresignedFile, 0, len(req.Files)),
// 	}

// 	now := time.Now()
//     // Базовая часть пути для всех файлов этого запроса
//     basePath := fmt.Sprintf("%d/%02d/%s/%s", now.Year(), now.Month(), roomID, req.PostId)

//     // 1. Обработка основных файлов поста
//     for _, file := range req.Files {
//         ext := filepath.Ext(file.Filename)
//         // Путь: year/month/room_id/post_id/file_id.ext
//         objectKey := fmt.Sprintf("%s/%s%s", basePath, file.UploadID, ext)

//         presignedURL, err := s.generatePresignedPutURL(ctx, objectKey, file.ContentType)
//         if err != nil {
//             return nil, fmt.Errorf("failed to generate URL for file %s: %w", file.UploadID, err)
//         }

//         publicURL := fmt.Sprintf("https://storage.yandexcloud.net/%s/%s", s.bucketMediaName, objectKey)

//         resp.Files = append(resp.Files, models.PresignedFile{
//             UploadID:     file.UploadID,
//             PresignedURL: presignedURL,
//             PublicURL:    publicURL,
//         })
//     }

//     // 2. Обработка превью поста
//     if req.Preview.PreviewId != "" {
//         // Для превью добавим префикс "preview_" или положим в папку "previews"
//         previewExt := filepath.Ext(req.Preview.PathPreview)
//         if previewExt == "" {
//             previewExt = ".jpg" // Дефолт для превью
//         }
        
// 		previewKey := fmt.Sprintf("%s/%s%s", basePath, req.Preview.PreviewId, previewExt)

//         // Контент-тип для превью обычно image/jpeg или image/webp
//         previewPresignedURL, err := s.generatePresignedPutURL(ctx, previewKey, "image/jpeg")
//         if err != nil {
//             return nil, fmt.Errorf("failed to generate URL for preview: %w", err)
//         }

//         previewPublicURL := fmt.Sprintf("https://storage.yandexcloud.net/%s/%s", s.bucketMediaName, previewKey)

//         resp.Preview = models.PreviewResponse{
//             PreviewReq:   req.Preview,
//             PresignedURL: previewPresignedURL,
//             PublicURL:    previewPublicURL,
//         }

// 		fmt.Println("Обработка превью прошла успешно")
//     }


// 	return resp, nil
// }

// func (s *PostService) GetCategoryIdBySlug(ctx context.Context, slug string) (int, error) {
//     if slug == "" {
//         return 0, fmt.Errorf("slug cannot be empty")
//     }

//     id, err := s.repo.GetCategoryIdBySlug(ctx, slug)
//     if err != nil {
//         // Логируем ошибку и пробрасываем выше
//         return -1, err
//     }

//     return id, nil
// }

// func (s *PostService) PublishPost(ctx context.Context, req models.PublishPostRequest) error {
// 	// Очистка хештегов: убираем пробелы, пустые строки и приводим к нижнему регистру
// 	var cleanHashtags []string
// 	for _, tag := range req.Hashtags {
// 		t := strings.TrimSpace(strings.ToLower(tag))
// 		if t != "" {
// 			cleanHashtags = append(cleanHashtags, t)
// 		}
// 	}
// 	req.Hashtags = cleanHashtags

// 	// Вызываем репозиторий. Транзакция будет управляться там.
// 	return s.repo.PublishPost(ctx, req)
// }

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
	objectKey := fmt.Sprintf("%s/%s/%s/%s", req.Post.RoomID, postID, req.Preview.UploadID, fileExtension)

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

func NewPostService(repository repositories.PostRepositoryInter, s3Client *s3.Client, bucketMediaName string) *PostService {
	return &PostService{repo: repository, s3Client: s3Client, bucketMediaName: bucketMediaName}
}