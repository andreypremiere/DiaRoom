package main

import (
	"account-microservice/contracts/account/requests"
	"account-microservice/repositories"
	"account-microservice/services"
	"account-microservice/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/andreypremiere/jwtmanager"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type App struct {
	// userService services.UserServiceInter
	accountService *services.AccountService
	// roomService services.RoomServiceInter
}

func (a *App) sendError(w http.ResponseWriter, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (a *App) newAccount(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Запрос должен содержать json данные", http.StatusUnsupportedMediaType)
        return
    }

    var newAccount requests.CreatingAccount
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields() 
    
    if err := decoder.Decode(&newAccount); err != nil {
        a.sendError(w, "Не удалось раскодировать тело запроса", http.StatusBadRequest)
        return
    }

    // 4. Бизнес-логика (создание аккаунта)
    newId, err := a.accountService.NewAccount(r.Context(), &newAccount)
    if err != nil {
        a.sendError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 5. Успешный ответ
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated) 
    json.NewEncoder(w).Encode(map[string]uuid.UUID{"userId": *newId})
}


func main() {
	// Создание фонового контекста для инициализации ресурсов
	ctx := context.Background()

	// Настройка пула соединений PostgreSQL
	user := os.Getenv("ACCOUNT_DB_USER")
	password := os.Getenv("ACCOUNT_DB_PASSWORD")
	host := os.Getenv("ACCOUNT_DB_HOST")
	port := os.Getenv("ACCOUNT_DB_PORT")
	db_name := os.Getenv("ACCOUNT_DB_NAMEBASE")

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", 
    user, password, host, port, db_name)
	poolPg, err := pgxpool.New(ctx, connString)
	if err != nil {
		fmt.Println("Ошибка инициализации пула соединений")
		return
	} else {
		fmt.Println("Пул соединений инициализирован")
	}

	// Настройка Redis
	hostRedis := os.Getenv("REDIS_OTP_HOST")
	portRedis := os.Getenv("REDIS_OTP_PORT")
	addrRedis := fmt.Sprintf("%s:%s", hostRedis, portRedis)
	rdb := redis.NewClient(&redis.Options{Addr: addrRedis})

	// Настройка email провайдера
	emailConfig := utils.EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Email:    os.Getenv("SMTP_EMAIL"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}

	emailProvider := utils.NewMailService(emailConfig)

	var newPasswordHasher *utils.PasswordHasher = utils.NewPasswordHasher(10)

	secretJwt := os.Getenv("JWT_SECRET")

	jwtmanager := jwtmanager.NewJWTManager(secretJwt, 15*time.Minute)

	accountRepo := repositories.NewAccountRepository(poolPg, rdb)

	accountService := services.NewAccountService(accountRepo, emailProvider, newPasswordHasher, jwtmanager)
	

	app := &App{
		accountService: accountService,
	}

	mux := http.NewServeMux()

    // Регистрация эндпоинтов с явным указанием методов
    mux.HandleFunc("POST /newAccount", app.newAccount)
    // mux.HandleFunc("POST /verifyUser/{userId}", app.verifyUserById) 
    // mux.HandleFunc("POST /login", app.LoginUser)
	// mux.HandleFunc("POST /repeatCode/{userId}", app.repeatSendingCode)
	// mux.HandleFunc("POST /refresh", app.refreshSession)
	// mux.HandleFunc("POST /logout", app.logout)

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