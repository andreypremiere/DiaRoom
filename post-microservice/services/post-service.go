package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"time"

	apperrors "post-microservice/app-errors"
	"post-microservice/clients"
	"post-microservice/contracts/responses"
	"post-microservice/models"
	"post-microservice/repositories"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type PostServiceInter interface {
    CreatePost(ctx context.Context, req models.CreatePostRequest) (*models.CreatePostResponse, error)
	GenerateMediaUrls(ctx context.Context, roomID uuid.UUID, req models.GenerateUrlsRequest) (*models.GenerateUrlsResponse, error)
	CreateAndAttachCanvas(ctx context.Context, postID uuid.UUID, payload json.RawMessage) error
	GetAllPosts(ctx context.Context) ([]responses.Post, error)
	GetPostForShowing(ctx context.Context, postId uuid.UUID) (*responses.ShowingPost, error)
	UpdateStatusUploaded(ctx context.Context, postID uuid.UUID) error
	GetPersonalPosts(ctx context.Context,  roomId uuid.UUID) ([]responses.PostInfoPersonal, error)
	GetRoomPosts(ctx context.Context,  roomId uuid.UUID) ([]responses.PostInfo, error)
	SyncViewsToDatabase(ctx context.Context)
	RecordView(ctx context.Context, postId uuid.UUID, userId uuid.UUID) error
}

type PostService struct {
	repo repositories.PostRepositoryInter
	s3Client   *s3.Client
	bucketMediaName string
	accountClient *clients.AccountClient
}

func (s *PostService) RecordView(ctx context.Context, postId uuid.UUID, roomId uuid.UUID) error {
    lockKey := fmt.Sprintf("view_lock:%s:%s", postId, roomId)
    
    // 1. Проверяем, смотрел ли пользователь этот пост за последние 2 часа
    alreadyViewed, err := s.repo.CheckView(ctx, lockKey)
    if err != nil {
        return err
    }
    
    if alreadyViewed > 0 {
        return nil 
    }

	s.repo.SetView(ctx, lockKey, "1", 2 * time.Hour)

	err = s.repo.HIncrView(ctx, postId.String())
	return err
}

func (s *PostService) SyncViewsToDatabase(ctx context.Context) {
    viewsMap, err := s.repo.GetAllViews(ctx)
    if err != nil || len(viewsMap) == 0 {
        return
    }

    toUpdate := make(map[string]int)
    for id, countStr := range viewsMap {
        if count, err := strconv.Atoi(countStr); err == nil {
            toUpdate[id] = count
        }
    }

    if err := s.repo.BulkIncrementViews(ctx, toUpdate); err != nil {
        log.Printf("Error syncing views: %v", err)
    }
}

func (s *PostService) GetRoomPosts(ctx context.Context,  roomId uuid.UUID) ([]responses.PostInfo, error) {
	posts, err := s.repo.GetRoomPosts(ctx, roomId)
	if err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return []responses.PostInfo{}, nil
	}

	for i := range posts {
		posts[i].PreviewUrl = fmt.Sprintf("https://storage.yandexcloud.net/%s", posts[i].PreviewUrl)
	}
	return posts, nil
}

func (s *PostService) GetPersonalPosts(ctx context.Context,  roomId uuid.UUID) ([]responses.PostInfoPersonal, error) {
	posts, err := s.repo.GetPersonalPosts(ctx, roomId)
	if err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return []responses.PostInfoPersonal{}, nil
	}

	for i := range posts {
		posts[i].PreviewUrl = fmt.Sprintf("https://storage.yandexcloud.net/%s", posts[i].PreviewUrl)
	}
	return posts, nil
}

func (s *PostService) GetPostForShowing(ctx context.Context, postId uuid.UUID) (*responses.ShowingPost, error) {
	post, err := s.repo.GetPostForShowing(ctx, postId)
	if err != nil {
		return nil, err
	}

	roomInfo, err := s.accountClient.GetAuthor(ctx, post.RoomId)
	if err != nil {
		return post, nil 
	}

	post.AvatarUrl = roomInfo.AvatarUrl
	post.RoomName = roomInfo.RoomName

	return post, nil 
}

func (s *PostService) GetAllPosts(ctx context.Context) ([]responses.Post, error) {
	postsInfo, err := s.repo.GetAllPosts(ctx)
	if err != nil {
		return nil, err
	}

	if len(postsInfo) == 0 {
		return []responses.Post{}, nil
	}

	for i := range postsInfo {
		postsInfo[i].PreviewUrl = fmt.Sprintf("https://storage.yandexcloud.net/%s", postsInfo[i].PreviewUrl)
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

func (s *PostService) UpdateStatusUploaded(ctx context.Context, postID uuid.UUID) error {
	err := s.repo.UpdateStatusUploaded(ctx, postID)
	if (err != nil) {
		fmt.Println("Ошибка обновления статуса для поста", err.Error())
		return err
	}

	err = s.repo.PushPostToQueue(ctx, postID)
	if (err != nil) {
		fmt.Println("Ошибка добавления id поста в очередь", err.Error())
		return err
	}
	return nil
}

func (s *PostService) CreateAndAttachCanvas(ctx context.Context, postID uuid.UUID, payload json.RawMessage) error {
	err := s.repo.InsertCanvasAndUpdatePost(ctx, postID, payload)
	if (err != nil) {
		fmt.Println("Ошибка добавления холста в пост", err.Error())
		return err
	}

	// err = s.repo.PushPostToQueue(ctx, postID)
	// if (err != nil) {
	// 	fmt.Println("Ошибка добавления id поста в очередь", err.Error())
	// 	return err
	// }
	return nil
}

func (s *PostService) CreatePost(ctx context.Context, req models.CreatePostRequest) (*models.CreatePostResponse, error) {
	// categoryID, err := s.repo.GetCategoryIdBySlug(ctx, req.Post.CategorySlug)
	// if err != nil {
    //     return nil, err
	// }

	postID, err := s.repo.CreatePost(ctx, models.CreatePostInternal{
		RoomID:     req.Post.RoomID,
		CategorySlug: req.Post.CategorySlug,
		Title:      req.Post.Title,
	})
	if err != nil {
		return nil, err
	}

    fileExtension := path.Ext(req.Preview.Filename)
    if fileExtension == "" {
		fileExtension = ".jpg" 
	}

	objectKey := fmt.Sprintf("%s/%s/%s%s", req.Post.RoomID, postID, req.Preview.UploadID, fileExtension)

	presignedURL, publicURL, err := s.generatePresignedAndPublicURL(ctx, objectKey, req.Preview.ContentType)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

    err = s.repo.UpdatePostPreviewURL(ctx, postID, publicURL)
    if err != nil {
        return nil, err
    }

    if len(req.Post.Hashtags) > 0 {
        cleanHashtags := CleanHashtags(req.Post.Hashtags)

		if err := s.repo.AddHashtagsToPost(ctx, postID, cleanHashtags); err != nil {
			fmt.Printf("Warning: failed to add hashtags to post %s: %v", postID, err)
		}
	}

	return &models.CreatePostResponse{
		PostID: postID,
		Preview: models.PreviewLinksResponse{
			PublicURL:    publicURL,
			PresignedURL: presignedURL,
		},
	}, nil

}

func (s *PostService) GenerateMediaUrls(ctx context.Context, roomID uuid.UUID, req models.GenerateUrlsRequest) (*models.GenerateUrlsResponse, error) {
	if len(req.Files) == 0 {
		return &models.GenerateUrlsResponse{
            Files: []models.GeneratedURL{},
        }, nil
	}

	response := &models.GenerateUrlsResponse{
		Files: make([]models.GeneratedURL, 0, len(req.Files)), 
	}

	for _, file := range req.Files {
		ext := path.Ext(file.Filename)
		if ext == "" {
			ext = ".jpg" 
		}

		objectKey := fmt.Sprintf("%s/%s/%s%s",
			roomID.String(),
			req.PostID.String(),
			file.UploadID.String(),
			ext,
		)

		presignedURL, publicURL, err := s.generatePresignedAndPublicURL(ctx, objectKey, file.ContentType)
		if err != nil {
			return nil, apperrors.ErrInternal
		}

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
		if clean != "" && clean != "#" {
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

func (s *PostService) generatePresignedAndPublicURL(ctx context.Context, objectKey string, contentType string) (presignedURL string, publicURL string, err error) {
	presigner := s3.NewPresignClient(s.s3Client)

	presignReq, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketMediaName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, func(o *s3.PresignOptions) {
		o.Expires = 15 * time.Minute
	})
	if err != nil {
		return "", "", apperrors.ErrGeneratingLinksForMedia
	}

	presignedURL = presignReq.URL
	publicURL = fmt.Sprintf("%s/%s", s.bucketMediaName, objectKey)

	return presignedURL, publicURL, nil
}

func NewPostService(repository repositories.PostRepositoryInter, s3Client *s3.Client, bucketMediaName string, accountClient *clients.AccountClient) *PostService {
	return &PostService{repo: repository, s3Client: s3Client, bucketMediaName: bucketMediaName, accountClient: accountClient}
}

