package main

import (
	// "context"
	// "database/sql"
	"context"
	"encoding/json"

	// "errors"
	"fmt"

	// "fmt"
	"net/http"
	"post-microservice/clients"
	"post-microservice/database"
	"post-microservice/models"
	"post-microservice/repositories"
	"post-microservice/services"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
)

// App содержит зависимости приложения, такие как сервисы бизнес-логики
type App struct {
	service services.PostServiceInter
}

func (a *App) sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (a *App) CreatePost(w http.ResponseWriter, r *http.Request) {
	var post models.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		a.sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	roomIDStr := r.Header.Get("X-Room-ID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, "Invalid Room ID format in header", http.StatusBadRequest)
		return
	}

	post.Post.RoomID = roomID
	fmt.Println("roomdId при создании поста: ", roomID)
	spew.Dump("CreatePostRequest при создании поста", post)

	result, err := a.service.CreatePost(r.Context(), post)
	if err != nil {
		a.sendError(w, "Ошибка при создании поста", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (a *App) GetPresignedUrls(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		a.sendError(w, "Missing X-Room-ID header", http.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, "Invalid Room ID format", http.StatusBadRequest)
		return
	}

	var req models.GenerateUrlsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	result, err := a.service.GenerateMediaUrls(r.Context(), roomID, req)
	if err != nil {
		fmt.Printf("Ошибка в GenerateMediaUrls: %v\n", err)
		a.sendError(w, "Failed to generate presigned URLs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (a *App) SaveCanvasHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем postId с помощью стандартного PathValue (Go 1.22+)
	postIDStr := r.PathValue("postId")
	if postIDStr == "" {
		a.sendError(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		a.sendError(w, "Invalid post ID format", http.StatusBadRequest)
		return
	}

	var req models.SaveCanvasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	err = a.service.CreateAndAttachCanvas(r.Context(), postID, req.Payload)
	if err != nil {
		a.sendError(w, "Failed to save canvas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}

func (a *App) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
        a.sendError(w, "Данный метод поддерживает только Get запросы", http.StatusMethodNotAllowed)
        return
    }

	posts, err := a.service.GetAllPosts(r.Context())
	if err != nil {
		a.sendError(w, "Ошибка получения всех постов" + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		a.sendError(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

func (a *App) GetPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
        a.sendError(w, "Данный метод поддерживает только Get запросы", http.StatusMethodNotAllowed)
        return
    }

	postIDStr := r.PathValue("postId")
	if postIDStr == "" {
		a.sendError(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		a.sendError(w, "Invalid post ID format", http.StatusBadRequest)
		return
	}

	post, err := a.service.GetPostForShowing(r.Context(), postID)
	if err != nil {
		a.sendError(w, "Ошибка получения всех постов" + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(post); err != nil {
		a.sendError(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

func main() {
	ctx := context.Background()

	fmt.Println("Инициализация клиента")
	s3Client, err := database.NewS3Client()
	if err != nil {
		fmt.Println("Ошибка создания клиента s3")
	}

	fmt.Println("Клиент s3 создан", s3Client)

	pool, err := database.InitPool(ctx)
	if err != nil {
		fmt.Println("Ошибка инициализации базы данных")
	}
	defer pool.Close() // Не забудь закрыть при выходе
	
	redisClient := database.InitRedisQueue()

	accountClient := clients.NewAccountClient("http://account-microservice:81")

	// Инициализация слоев приложения (Repository -> Service -> App)
	repository := repositories.NewPostRepository(pool, redisClient)
	service := services.NewPostService(repository, s3Client, "media-for-publication", accountClient)

	app := App{service: service}
	
	mux := http.NewServeMux()

	// Регистрация маршрутов с указанием метода 
	mux.HandleFunc("POST /createPost", app.CreatePost)
	mux.HandleFunc("POST /getPresignedUrls", app.GetPresignedUrls)
	mux.HandleFunc("POST /{postId}/canvas", app.SaveCanvasHandler)
	mux.HandleFunc("GET /allPosts", app.GetAllPosts)
	mux.HandleFunc("GET /getPost/{postId}", app.GetPost)

	fmt.Println("Сервер постов запущен на порту :81")

	if err := http.ListenAndServe(":81", mux); err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}