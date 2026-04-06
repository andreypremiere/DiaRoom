package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"user-microservice/contracts/requests"
	"user-microservice/repositories"
	"user-microservice/services"
	"user-microservice/utils"

	"github.com/andreypremiere/jwtmanager"
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

func (a *App) repeatSendingCode(w http.ResponseWriter, r *http.Request) {
	// 1. Проверка метода
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    userIDStr := r.PathValue("userId")
    if userIDStr == "" {
        a.sendError(w, "ID пользователя не указан", http.StatusBadRequest)
        return
    }

    // Парсим строку в uuid (если ты используешь uuid в базе)
    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        a.sendError(w, "Некорректный формат UUID", http.StatusBadRequest)
        return
    }

	// Вызов сервиса для повторной отправки кода
	err = a.userService.RepeatSendingCode(r.Context(), userID)
	if err != nil {
		a.sendError(w, err.Error(), http.StatusBadRequest)
        return
	}

	w.WriteHeader(http.StatusOK)
}

// verifyUserById проверяет код подтверждения и выдает JWT токен
func (a *App) verifyUserById(w http.ResponseWriter, r *http.Request) {
    // 1. Проверка метода
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    // 2. Извлекаем userId из пути (Path Value)
    userIDStr := r.PathValue("userId")
    if userIDStr == "" {
        a.sendError(w, "ID пользователя не указан", http.StatusBadRequest)
        return
    }

    // Парсим строку в uuid (если ты используешь uuid в базе)
    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        a.sendError(w, "Некорректный формат UUID", http.StatusBadRequest)
        return
    }

    var userVerify requests.VerifyUserById

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&userVerify); err != nil {
		// Сработает, если прислали лишнее поле или невалидный JSON
		a.sendError(w, "Тело запроса содержит недопустимые поля или неверный формат", http.StatusBadRequest)
		return
	}

    // 4. Подготавливаем модель для сервиса
    userVerify.UserId = userID

    // 5. Вызов сервиса
    response, err := a.userService.VerifyCode(r.Context(), userVerify)
    if err != nil {
        // Если код неверный, лучше вернуть 401 Unauthorized или 400
        a.sendError(w, "Ошибка верификации: " + err.Error(), http.StatusBadRequest)
        return
    }

    // 6. Успешный ответ
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) refreshSession(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
        return
    }

    var request struct {
        RefreshToken string `json:"refreshToken"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        a.sendError(w, "Некорректное тело запроса", http.StatusBadRequest)
        return
    }

    // Вызов сервиса для ротации токенов
    response, err := a.userService.RefreshSession(r.Context(), request.RefreshToken)
    if err != nil {
        // Если токен старый или его нет — 401, чтобы Flutter разлогинил юзера
        a.sendError(w, "Сессия недействительна: "+err.Error(), http.StatusUnauthorized)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
        return
    }

    var request struct {
        RefreshToken string `json:"refreshToken"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        a.sendError(w, "Некорректное тело запроса", http.StatusBadRequest)
        return
    }

    err := a.userService.Logout(r.Context(), request.RefreshToken)
    if err != nil {
        a.sendError(w, "Ошибка при удалении сессии", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent) // 204 — успешно, тела нет
}

func (a *App) LoginUser(w http.ResponseWriter, r *http.Request) {
    // 1. Проверка метода
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    // 2. Валидация заголовка
    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Тип данных не поддерживается", http.StatusUnsupportedMediaType)
        return
    }

    // 3. Декодирование
    var userLogin requests.UserLogin
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields() 

    if err := decoder.Decode(&userLogin); err != nil {
        a.sendError(w, "Не удалось раскодировать тело запроса или найдены лишние поля", http.StatusBadRequest)
        return
    }

    userActive, userInactive, err := a.userService.LoginUser(r.Context(), userLogin)

	if userActive == nil && err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"isActivated": false, "userData": userInactive})
		return
	}

    if err != nil {
        a.sendError(w, "Ошибка при входе: " + err.Error(), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]any{"isActivated": true, "userData": userActive})
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

	secretJwt := os.Getenv("JWT_SECRET")

	jwtmanager := jwtmanager.NewJWTManager(secretJwt, 15*time.Minute)
	// Сборка графа зависимостей приложения
	userRepo := repositories.NewUserRepository(poolPg, rdb)
	userServ := services.NewUserService(userRepo, emailProvider, newPasswordHasher, jwtmanager)

	// Внедрение сервисов в структуру приложения
	app := &App{
		userService: userServ,
	}

	mux := http.NewServeMux()

    // Регистрация эндпоинтов с явным указанием методов
    mux.HandleFunc("POST /newUser", app.newUser)
    mux.HandleFunc("POST /verifyUser/{userId}", app.verifyUserById) 
    mux.HandleFunc("POST /login", app.LoginUser)
	mux.HandleFunc("POST /repeatCode/{userId}", app.repeatSendingCode)
	mux.HandleFunc("POST /refresh", app.refreshSession)
	mux.HandleFunc("POST /logout", app.logout)

	fmt.Println("Сервер запущен на :81")
	// Запуск веб-сервера на порту 81
	if err := http.ListenAndServe(":81", mux); err != nil {
		fmt.Println(err.Error())
	}

	// Гарантированное закрытие пула соединений при завершении программы
	defer func() {
		poolPg.Close()
	}()
}