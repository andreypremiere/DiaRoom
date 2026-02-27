package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"user-microservice/models"
	"user-microservice/repositories"
	"user-microservice/services"
	"user-microservice/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/andreypremiere/jwtmanager"
)

// App объединяет все зависимости нашего приложения
type App struct {
	userService services.UserServiceInter
}

// newUser теперь является методом структуры App
func (a *App) newUser(w http.ResponseWriter, r *http.Request) {
	// Проверка, что это метод POST
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		// Сериализуем мапу в json
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

	// Создаем экземпляр пользователя для регистрации
	var newUser = models.RegisterUser{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&newUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не удалось раскодировать тело запроса"})
		return
	}

	// Отправляем данные в сервис
	newId, err := a.userService.AddUser(r.Context(), &newUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка создания пользователя на стороне сервера"})
		// fmt.Println("Ошибка:", err.Error())		
		return
	}

	// Отправка ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"id": newId})
}

func (a *App) verifyUserById(w http.ResponseWriter, r *http.Request) {
	// Проверка на тип запроса
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Эндпоинт поддерживает только POST запросы"})
		return
	}

	// Проверка на тип данных тела запроса
	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Тип данных не поддерживается"})
		return
	}

	userVerify := models.VerifyUserById{}
	err := json.NewDecoder(r.Body).Decode(&userVerify)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при сериализации данных с клиента"})
		return
	}

	_, err2 := a.userService.VerifyCode(r.Context(), userVerify)
	if err2 != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка на стороне сервера"})
		return
	}

	jwtmanager := jwtmanager.NewJWTManager("1111", 2*time.Minute)
	token, err := jwtmanager.Generate(userVerify.UserId.String(), "user")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка создания токена"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"jwt": token})
}

func main() {
	// Создание базового контекста
	ctx := context.Background()

	// Перенести функции инициализации базы данных в database-clients.go

	// Создание пула соединений для базы данных postgres
	connString := "postgres://postgres:1111@postgresql-users:5432/db_users"
	poolPg, err := pgxpool.New(ctx, connString)
	if err != nil {
		fmt.Println("Ошибка инициализации пула соединений")
		return
	}

	// Создание клиента для redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis-cache:6379",
		Password: "",
		DB: 0,
	})

	smsProvider := utils.ConsoleSms{}

	// Создание зависимостей
	userRepo := repositories.NewUserRepository(poolPg, rdb)
	userServ := services.NewUserService(userRepo, smsProvider)

	// Создание структуры сервера и внедрение зависимостей
	app := &App{
		userService: userServ,
	}

	// Регистрируем маршруты
	http.HandleFunc("/newUser", app.newUser)
	http.HandleFunc("/verifyUser", app.verifyUserById)

	// Запускаем сервер
	fmt.Println("Сервер запущен на :81")
	if err := http.ListenAndServe(":81", nil); err != nil {
		fmt.Println(err.Error())
	}

	// Закрытие соединений
	defer func () {
		poolPg.Close()
	} ()
}
