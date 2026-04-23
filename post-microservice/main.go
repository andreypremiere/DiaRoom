package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	apperrors "post-microservice/app-errors"
	"post-microservice/clients"
	"post-microservice/database"
	"post-microservice/models"
	"post-microservice/repositories"
	"post-microservice/services"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type App struct {
	service services.PostServiceInter
}

func (a *App) sendError(w http.ResponseWriter, err error) {
    var appErr apperrors.AppError
    
    status := http.StatusInternalServerError
    
    if errors.As(err, &appErr) {
        switch appErr.Code {
        case "NOT_FOUND":
            status = http.StatusNotFound
        case "ALREADY_EXISTS":
            status = http.StatusConflict
        case "METHOD_NOT_ALLOWED":
            status = http.StatusMethodNotAllowed 
        case "UNSUPPORTED_TYPE":
            status = http.StatusUnsupportedMediaType 
        case "INVALID_INPUT":
            status = http.StatusBadRequest
        default:
            status = http.StatusBadRequest
        }
    } else {
        appErr = apperrors.ErrInternal
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error_code": appErr.Code,
        "message":    appErr.Message,
    })
}

func (a *App) CreatePost(w http.ResponseWriter, r *http.Request) {
    var post models.CreatePostRequest
    if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    roomIDStr := r.Header.Get("X-Room-ID")
    roomID, err := uuid.Parse(roomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    post.Post.RoomID = roomID
    result, err := a.service.CreatePost(r.Context(), post)
    if err != nil {
        a.sendError(w, err) 
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func (a *App) GetPresignedUrls(w http.ResponseWriter, r *http.Request) {
    roomIDStr := r.Header.Get("X-Room-ID")
    if roomIDStr == "" {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    roomID, err := uuid.Parse(roomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    var req models.GenerateUrlsRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }
    defer r.Body.Close()

    result, err := a.service.GenerateMediaUrls(r.Context(), roomID, req)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(result)
}

func (a *App) SaveCanvasHandler(w http.ResponseWriter, r *http.Request) {
    postIDStr := r.PathValue("postId")
    if postIDStr == "" {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    postID, err := uuid.Parse(postIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    var req models.SaveCanvasRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    err = a.service.CreateAndAttachCanvas(r.Context(), postID, req.Payload)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "success"}`))
}

func (a *App) UpdateStatusUploaded(w http.ResponseWriter, r *http.Request) {
    postIDStr := r.PathValue("postId")
    if postIDStr == "" {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    postID, err := uuid.Parse(postIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    err = a.service.UpdateStatusUploaded(r.Context(), postID)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func (a *App) GetAllPosts(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    posts, err := a.service.GetAllPosts(r.Context())
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(posts)
}

func (a *App) GetPost(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    postIDStr := r.PathValue("postId")
    postID, err := uuid.Parse(postIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    post, err := a.service.GetPostForShowing(r.Context(), postID)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(post)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

	// S3 Client
    s3Client, err := database.NewS3Client()
    if err != nil {
        log.Fatalf("Критическая ошибка: не удалось создать клиент S3: %v", err)
    }

    // Database Pool
    pool, err := database.InitPool(ctx)
    if err != nil {
        log.Fatalf("Критическая ошибка: не удалось инициализировать БД: %v", err)
    }
    defer pool.Close() 

    // Redis Client
    redisClient := database.InitRedisQueue()
    defer redisClient.Close() 

	accountClient := clients.NewAccountClient("http://account-microservice:81")

	repository := repositories.NewPostRepository(pool, redisClient)
	service := services.NewPostService(repository, s3Client, "media-for-publication", accountClient)

	app := App{service: service}
	
	mux := http.NewServeMux()

	mux.HandleFunc("POST /createPost", app.CreatePost)
	mux.HandleFunc("POST /updateStatusUploaded/{postId}", app.UpdateStatusUploaded)
	mux.HandleFunc("POST /getPresignedUrls", app.GetPresignedUrls)
	mux.HandleFunc("POST /saveCanvas/{postId}", app.SaveCanvasHandler)
	mux.HandleFunc("GET /allPosts", app.GetAllPosts)
	mux.HandleFunc("GET /getPost/{postId}", app.GetPost)

	server := &http.Server{
        Addr:    ":81",
        Handler: mux,
    }

	go func() {
        fmt.Println("Сервер постов запущен на порту :81")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Ошибка сервера: %v", err)
        }
    }()

    <-ctx.Done()
    fmt.Println("Завершение работы...")
    
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(shutdownCtx); err != nil {
        fmt.Printf("Ошибка при остановке сервера: %v\n", err)
    }
}