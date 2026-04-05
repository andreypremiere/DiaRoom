package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"user-microservice/contracts/requests"
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

func (a *App) sendError(w http.ResponseWriter, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// newUser обрабатывает регистрацию нового пользователя через POST запрос
func (a *App) newUser(w http.ResponseWriter, r *http.Request) {
    // 1. Проверка метода
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    // 2. Валидация заголовка типа контента
    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Тип данных не поддерживается", http.StatusUnsupportedMediaType)
        return
    }

    // 3. Декодирование JSON
    var newUser requests.UserCreatingContract
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields() // Строгая валидация (проверяет только на лишние поля)
    
    if err := decoder.Decode(&newUser); err != nil {
        a.sendError(w, "Не удалось раскодировать тело запроса", http.StatusBadRequest)
        return
    }

    // 4. Бизнес-логика (создание пользователя)
    newId, err := a.userService.AddUser(r.Context(), &newUser)
    if err != nil {
        a.sendError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 5. Успешный ответ
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated) 
    json.NewEncoder(w).Encode(map[string]uuid.UUID{"userId": newId})
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
	emailConfig := utils.EmailConfig{
		Host:     "smtp.yandex.ru",
		Port:     "587",
		Email:    os.Getenv("SMTP_EMAIL"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}

	emailProvider := utils.NewMailService(emailConfig)

	var newPasswordHasher *utils.PasswordHasher = utils.NewPasswordHasher(10)

	// Сборка графа зависимостей приложения
	userRepo := repositories.NewUserRepository(poolPg, rdb)
	userServ := services.NewUserService(userRepo, emailProvider, newPasswordHasher)

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