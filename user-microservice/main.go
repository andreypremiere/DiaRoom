package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"user-microservice/models"
	"user-microservice/repositories"
	"user-microservice/services"
	"user-microservice/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// App объединяет все зависимости нашего приложения
type App struct {
	userService services.UserServiceInter
}

// newUser обрабатывает регистрацию нового пользователя через POST запрос
func (a *App) newUser(w http.ResponseWriter, r *http.Request) {
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

	var newUser = models.RegisterUser{}
	decoder := json.NewDecoder(r.Body)
	// Запрет неизвестных полей в JSON для строгой валидации
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&newUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Не удалось раскодировать тело запроса"})
		return
	}

	// Передача данных в слой бизнес-логики
	newId, err := a.userService.AddUser(r.Context(), &newUser)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка создания пользователя на стороне сервера"})
		return
	}

	// Успешный ответ с идентификатором созданного пользователя
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"id": newId})
}

// verifyUserById проверяет код подтверждения и выдает JWT токен
func (a *App) verifyUserById(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Эндпоинт поддерживает только POST запросы"})
		return
	}

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

	// Вызов сервиса для верификации кода
	token, err2 := a.userService.VerifyCode(r.Context(), userVerify)
	if err2 != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка на стороне сервера" + err2.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// LoginUser инициирует процесс входа пользователя в систему
func (a *App) LoginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Эндпоинт поддерживает только POST запросы"})
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]string{"error": "Тип данных не поддерживается"})
		return
	}

	var input struct {
		Value string `json:"value"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при сериализации данных с клиента"})
		return
	}

	// Поиск пользователя и отправка SMS кода
	err, userId := a.userService.LoginUser(r.Context(), input.Value)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка при выполнении запроса"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]uuid.UUID{"userId": userId})
}

func main() {
	// Создание фонового контекста для инициализации ресурсов
	ctx := context.Background()

	// Настройка пула соединений PostgreSQL
	connString := "postgres://postgres:1111@postgresql-users:5432/db_users"
	poolPg, err := pgxpool.New(ctx, connString)
	if err != nil {
		fmt.Println("Ошибка инициализации пула соединений")
		return
	}

	// Настройка клиента для кэширования в Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-cache:6379",
		Password: "",
		DB:       0,
	})

	// Использование консольного SMS провайдера для разработки
	smsProvider := utils.ConsoleSms{}

	// Сборка графа зависимостей приложения
	userRepo := repositories.NewUserRepository(poolPg, rdb)
	userServ := services.NewUserService(userRepo, smsProvider)

	// Внедрение сервисов в структуру приложения
	app := &App{
		userService: userServ,
	}

	// Регистрация эндпоинтов в стандартном мультиплексоре
	http.HandleFunc("/newUser", app.newUser)
	http.HandleFunc("/verifyUser", app.verifyUserById)
	http.HandleFunc("/login", app.LoginUser)

	fmt.Println("Сервер запущен на :81")
	// Запуск веб-сервера на порту 81
	if err := http.ListenAndServe(":81", nil); err != nil {
		fmt.Println(err.Error())
	}

	// Гарантированное закрытие пула соединений при завершении программы
	defer func() {
		poolPg.Close()
	}()
}