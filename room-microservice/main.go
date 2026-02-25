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

type App struct {
	// Переменные соединений
	roomService services.RoomServiceInter
}

func (a *App) newRoom(w http.ResponseWriter, r *http.Request) {
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

	newRoom := models.BaseRoom{}
	err := json.NewDecoder(r.Body).Decode(&newRoom)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка преобразования тела запроса"})
		return
	}

	idRoom, err := a.roomService.AddRoom(r.Context(), &newRoom)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка создания комнаты на стороне сервера"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"idRoom": idRoom})
}

func main() {
	ctx := context.Background()

	connString := "postgres://postgres:1111@postgresql-rooms:5432/db_rooms"
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		fmt.Println("Не удалось создать пул соединений для базы данных")
		return
	}

	roomRepo := repositories.NewRoomRepository(db)
	roomServ := services.NewRoomService(roomRepo)

	app := App{roomServ}
	
	http.HandleFunc("/newRoom", app.newRoom)

	http.ListenAndServe(":81", nil)
}