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
	"post-microservice/database"
	"post-microservice/models"
	"post-microservice/repositories"
	"post-microservice/services"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/go-chi/chi"
)

// App содержит зависимости приложения, такие как сервисы бизнес-логики
type App struct {
	service services.PostServiceInter
}


func (h *App) CreatePost(w http.ResponseWriter, r *http.Request) {
    var post models.CreatePostRequest
    if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
        http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
        return
    }

	roomIDStr := r.Header.Get("X-Room-ID")

	roomID, err := uuid.Parse(roomIDStr)
    if err != nil {
        http.Error(w, "Invalid Room ID format", http.StatusBadRequest)
        return
    }

	post.Post.RoomID = roomID  // Задаем roomId из заголовка

	fmt.Println("roomdId при создании поста: ", roomID)
	spew.Dump("CreatePostRequest при создании поста", post)

    result, err := h.service.CreatePost(r.Context(), post)
    if err != nil {
		http.Error(w, `{"error": "Ошибка при создании"}`, http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func (h *App) GetPresignedUrls(w http.ResponseWriter, r *http.Request) {
	// 1. Получаем и валидируем RoomID из заголовка
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		http.Error(w, `{"error": "Missing X-Room-ID header"}`, http.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid Room ID format"}`, http.StatusBadRequest)
		return
	}

	// 2. Декодируем тело запроса
	var req models.GenerateUrlsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 3. Вызываем сервисный слой
	result, err := h.service.GenerateMediaUrls(r.Context(), roomID, req)
	if err != nil {
		// Логируем реальную ошибку на сервере
		fmt.Printf("Ошибка в GenerateMediaUrls: %v\n", err)
		http.Error(w, `{"error": "Failed to generate presigned URLs"}`, http.StatusInternalServerError)
		return
	}

	// 4. Отправляем успешный ответ клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *App) SaveCanvasHandler(w http.ResponseWriter, r *http.Request) {
	// Достаем ID поста из URL (например, используем chi или gorilla/mux)
	postIDStr := chi.URLParam(r, "postId")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var req models.SaveCanvasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Вызываем сервис
	err = h.service.CreateAndAttachCanvas(r.Context(), postID, req.Payload)
	if err != nil {
		// В реальном проекте здесь стоит разделять ошибки (404, 500 и т.д.)
		http.Error(w, "Failed to save canvas", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
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

	// Инициализация слоев приложения (Repository -> Service -> App)
	repository := repositories.NewPostRepository(pool, redisClient)
	service := services.NewPostService(repository, s3Client, "media-for-publication")

	app := App{service: service}
	
	r := chi.NewRouter()

    // Обычные маршруты
    r.Post("/createPost", app.CreatePost)
    r.Post("/getPresignedUrls", app.GetPresignedUrls)

    // Маршрут с параметром {postId}
    // Путь будет: /post/{postId}/canvas
    r.Post("/{postId}/canvas", app.SaveCanvasHandler)

    http.ListenAndServe(":81", r) // Передаем 'r' вместо 'nil'
}