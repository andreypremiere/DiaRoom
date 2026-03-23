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
)

// App содержит зависимости приложения, такие как сервисы бизнес-логики
type App struct {
	service services.PostServiceInter
	// roomService services.RoomServiceInter

}

// newRoom обрабатывает HTTP-запрос на создание новой комнаты
func (a *App) getPresignedUrls(w http.ResponseWriter, r *http.Request) {
	// Проверка разрешенного HTTP метода
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Данный метод поддерживает только POST запросы"})
		return
	}

	// Валидация заголовка типа контента	
	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Тип данных не поддерживается"})
		return
	}

    roomID := r.Header.Get("X-Room-ID")

	var req models.PresignedRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, `{"error": "Ошибка чтения JSON"}`, http.StatusBadRequest)
		return
	}

	fmt.Println("Принятный из заголовока RoomId:" + roomID)

	spew.Dump(req)
	
	resp, err := a.service.GetPresignedUrls(r.Context(), &req, roomID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	spew.Dump(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}

func (h *App) CreatePost(w http.ResponseWriter, r *http.Request) {
    var req models.CreatePostRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
        return
    }

	roomIDStr := r.Header.Get("X-Room-ID")

	roomID, err := uuid.Parse(roomIDStr)
    if err != nil {
        // Если пришла невалидная строка (например, "null" или просто текст)
        http.Error(w, "Invalid Room ID format", http.StatusBadRequest)
        return
    }

	req.RoomID = roomID

	spew.Dump(req)


    postID, err := h.service.CreatePost(r.Context(), req)
    if err != nil {
		http.Error(w, `{"error": "Ошибка при создании"}`, http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(models.CreatePostResponse{
        PostID: postID,
        Status: "processing",
    })
}

func (h *App) PublishPost(w http.ResponseWriter, r *http.Request) {
	var req models.PublishPostRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	// Базовая валидация
	if req.PostID == uuid.Nil {
		http.Error(w, `{"error": "postId is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Payload) == 0 || string(req.Payload) == "null" {
		http.Error(w, `{"error": "payload is required"}`, http.StatusBadRequest)
		return
	}

	// Передаем в сервис
	err := h.service.PublishPost(r.Context(), req)
	if err != nil {
		http.Error(w, `{"error": "Failed to publish post"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.PublishPostResponse{
		Message: "Post published successfully",
		Status:  "published",
	})
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
	

	// Инициализация слоев приложения (Repository -> Service -> App)
	repository := repositories.NewPostRepository(pool)
	service := services.NewPostService(repository, s3Client, "media-for-publication")


	app := App{service: service}
	
	// Регистрация маршрутов
	http.HandleFunc("/getPresignedUrls", app.getPresignedUrls)
	http.HandleFunc("/createPost", app.CreatePost)
	http.HandleFunc("/publishPost", app.PublishPost)


	http.ListenAndServe(":81", nil)
}