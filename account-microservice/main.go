package main

import (
	"account-microservice/contracts/account/requests"
	"account-microservice/contracts/account/responses"
	"account-microservice/database"
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

    newId, err := a.accountService.NewAccount(r.Context(), &newAccount)
    if err != nil {
        a.sendError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated) 
    json.NewEncoder(w).Encode(map[string]uuid.UUID{"userId": *newId})
}

func (a *App) verify(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Запрос должен содержать json данные", http.StatusUnsupportedMediaType)
        return
    }

    userIDStr := r.PathValue("userId")
    if userIDStr == "" {
        a.sendError(w, "ID пользователя не указан", http.StatusBadRequest)
        return
    }

    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        a.sendError(w, "Некорректный формат UUID", http.StatusBadRequest)
        return
    }

    var userVerify requests.VerifyUser

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&userVerify); err != nil {
		a.sendError(w, "Тело запроса содержит недопустимые поля или неверный формат", http.StatusBadRequest)
		return
	}

    userVerify.UserId = userID

    response, err := a.accountService.VerifyCode(r.Context(), &userVerify)

    if err != nil {
		if err.Error() == "user did not confirm the email" {
			a.sendError(w, err.Error(), http.StatusForbidden)
			return
		}
		if err.Error() == "couldn't update status" {
			a.sendError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err.Error() == "roomId search error" {
			a.sendError(w, err.Error(), http.StatusNotFound)
			return
		}
        a.sendError(w, "Ошибка верификации: " + err.Error(), http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) LoginUser(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Запрос должен содержать json данные", http.StatusUnsupportedMediaType)
        return
    }

	var loginUser requests.LoginUser

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&loginUser); err != nil {
		a.sendError(w, "Тело запроса содержит недопустимые поля или неверный формат", http.StatusBadRequest)
		return
	}

	user, err := a.accountService.LoginUser(r.Context(), &loginUser)
	if err != nil {
		a.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(responses.LoginResponse{UserId: user.ID, Email: user.Email})
}

func (a *App) repeatCode(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

	userIDStr := r.PathValue("userId")
    if userIDStr == "" {
        a.sendError(w, "ID пользователя не указан", http.StatusBadRequest)
        return
    }

    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        a.sendError(w, "Некорректный формат UUID", http.StatusBadRequest)
        return
    }

	err = a.accountService.RepeatSendingCode(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) refreshSession(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Запрос должен содержать json данные", http.StatusUnsupportedMediaType)
        return
    }

    var request struct {
        RefreshToken string `json:"refreshToken"`
    }

    decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&request); err != nil {
		a.sendError(w, "Тело запроса содержит недопустимые поля или неверный формат", http.StatusBadRequest)
		return
	}

	// Если токен истек или его нет
    response, err := a.accountService.RefreshSession(r.Context(), request.RefreshToken)
    if err != nil {
        a.sendError(w, "Сессия недействительна: "+err.Error(), http.StatusBadRequest)
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

	if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Запрос должен содержать json данные", http.StatusUnsupportedMediaType)
        return
    }

    var request struct {
        RefreshToken string `json:"refreshToken"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        a.sendError(w, "Некорректное тело запроса", http.StatusBadRequest)
        return
    }

    err := a.accountService.Logout(r.Context(), request.RefreshToken)
    if err != nil {
        a.sendError(w, "Ошибка при удалении сессии", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (a *App) getRoom(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        a.sendError(w, "Данный метод поддерживает только Get запросы", http.StatusMethodNotAllowed)
        return
    }

	roomIDStr := r.PathValue("roomId")
    if roomIDStr == "" {
        a.sendError(w, "ID комнаты не указан", http.StatusBadRequest)
        return
    }

    roomId, err := uuid.Parse(roomIDStr)
    if err != nil {
        a.sendError(w, "Некорректный формат UUID", http.StatusBadRequest)
        return
    }

    room, err := a.accountService.GetRoom(r.Context(), roomId)
    if err != nil {
        a.sendError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(room)
}

func (a *App) updateRoom(w http.ResponseWriter, r *http.Request) {
    // 1. Проверка метода
    if r.Method != http.MethodPost {
        a.sendError(w, "Данный метод поддерживает только POST запросы", http.StatusMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, "Запрос должен содержать json данные", http.StatusUnsupportedMediaType)
        return
    }

    // Извлекаем данные из заголовков
    roomID := r.Header.Get("X-Room-ID")

    roomId, err := uuid.Parse(roomID)
    if err != nil {
        a.sendError(w, "Некорректный формат UUID", http.StatusBadRequest)
        return
    }

    // Декодируем тело запроса
    var request requests.UpdateRoomRequest // Твоя структура с json тегами
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()

    if err := decoder.Decode(&request); err != nil {
        a.sendError(w, "Тело запроса содержит недопустимые поля или неверный формат", http.StatusBadRequest)
        return
    }

    response, err := a.accountService.UpdateRoom(r.Context(), roomId, &request)
    if err != nil {
        a.sendError(w, "Не удалось обновить данные: " + err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) getRoomsInfo(w http.ResponseWriter, r *http.Request) {
    // 1. Только POST
    if r.Method != http.MethodPost {
        a.sendError(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
        return
    }

    // 2. Декодируем список ID
    var req requests.GetRoomsBatch
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, "Некорректный JSON", http.StatusBadRequest)
        return
    }

    // 3. Вызов сервиса
    roomsMap, err := a.accountService.GetRoomsInfoBatch(r.Context(), req.UserIDs)
    if err != nil {
        a.sendError(w, "Ошибка при получении данных комнат", http.StatusInternalServerError)
        return
    }

    // 4. Отправка мапы map[uuid]RoomInfo
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(roomsMap)
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

    // Настройка S3 клиента
    s3Client := database.InitS3Client()

    s3Manager := services.NewS3Manager(s3Client, "avatars-diaroom-1")

	emailProvider := utils.NewMailService(emailConfig)

	var newPasswordHasher *utils.PasswordHasher = utils.NewPasswordHasher(10)

	secretJwt := os.Getenv("JWT_SECRET")

	jwtmanager := jwtmanager.NewJWTManager(secretJwt, 2*time.Minute)

	accountRepo := repositories.NewAccountRepository(poolPg, rdb)

	accountService := services.NewAccountService(accountRepo, emailProvider, newPasswordHasher, jwtmanager, s3Manager)
	

	app := &App{
		accountService: accountService,
	}

	mux := http.NewServeMux()

    mux.HandleFunc("POST /newAccount", app.newAccount)
    mux.HandleFunc("POST /verify/{userId}", app.verify) 
    mux.HandleFunc("POST /login", app.LoginUser)
	mux.HandleFunc("POST /repeatCode/{userId}", app.repeatCode)
	mux.HandleFunc("POST /refreshSession", app.refreshSession)
	mux.HandleFunc("POST /logout", app.logout)
    mux.HandleFunc("GET /room/{roomId}", app.getRoom)
    mux.HandleFunc("POST /updateRoom", app.updateRoom)
    mux.HandleFunc("POST /getRoomsInfo", app.getRoomsInfo)

	fmt.Println("Сервер запущен на :81")
	if err := http.ListenAndServe(":81", mux); err != nil {
		fmt.Println(err.Error())
	}

	defer func() {
		poolPg.Close()
	}()
}