package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"room-microservice/models"
	"room-microservice/repositories"
	"room-microservice/services"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// App содержит зависимости приложения, такие как сервисы бизнес-логики
type App struct {
	roomService services.RoomServiceInter
}

// newRoom обрабатывает HTTP-запрос на создание новой комнаты
func (a *App) newRoom(w http.ResponseWriter, r *http.Request) {
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

	newRoom := models.BaseRoom{}

	// Декодирование JSON из тела запроса в структуру
	err := json.NewDecoder(r.Body).Decode(&newRoom)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка преобразования тела запроса"})
		return
	}

	// Вызов сервиса для создания комнаты
	idRoom, err := a.roomService.AddRoom(r.Context(), &newRoom)

	
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка создания комнаты на стороне сервера"})
		return
	}

	// Отправка успешного ответа с ID созданной комнаты
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"idRoom": idRoom})
}

// getRoomIdByUserId обрабатывает запрос на поиск ID комнаты по пользователю
func (a *App) getRoomIdByUserId(w http.ResponseWriter, r *http.Request) {
	// Проверка разрешенного HTTP метода
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Данный метод поддерживает только POST запросы"})
		return
	}

	// Проверка на тип данных тела запроса
	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Тип данных не поддерживается"})
		return
	}

	var userId struct {UserId uuid.UUID `json:"userId"`}

	// Декодирование тела запроса
	err := json.NewDecoder(r.Body).Decode(&userId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка преобразования тела запроса"})
		return
	}

	// Получение ID комнаты через сервис
	roomId, err := a.roomService.GetRoomIdByUserId(r.Context(), userId.UserId)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка на стороне сервера" + err.Error()})
		return
	}

	// Отправка успешного ответа 
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"roomId": roomId})
}

// getRoomByRoomId возвращает расширенные данные комнаты по её ID
func (a *App) getRoomByRoomId(w http.ResponseWriter, r *http.Request) {
	// Проверка разрешенного HTTP метода
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Данный метод поддерживает только POST запросы"})
		return
	}

	// Проверка на тип данных тела запроса
	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Тип данных не поддерживается"})
		return
	}

	var roomId struct {RoomId uuid.UUID `json:"roomId"`}

	// Декодирование ответа
	err := json.NewDecoder(r.Body).Decode(&roomId)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка преобразования тела запроса"})
		return
	}

	// Получение полной информации о комнате через сервис
	room, err := a.roomService.GetRoomByRoomId(r.Context(), roomId.RoomId)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка на стороне сервера" + err.Error()})
		return
	}

	// Отправка успешного ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(room)
}

func main() {
	ctx := context.Background()

	// Настройка подключения к PostgreSQL
	connString := "postgres://postgres:1111@postgresql-rooms:5432/db_rooms"
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		fmt.Println("Не удалось создать пул соединений для базы данных")
		return
	}

	// Инициализация слоев приложения (Repository -> Service -> App)
	roomRepo := repositories.NewRoomRepository(db)
	roomServ := services.NewRoomService(roomRepo)

	app := App{roomServ}
	
	// Регистрация маршрутов
	http.HandleFunc("/newRoom", app.newRoom)
	http.HandleFunc("/getRoomIdByUserId", app.getRoomIdByUserId)
	http.HandleFunc("/getRoomByRoomId", app.getRoomByRoomId)

	http.ListenAndServe(":81", nil)
}