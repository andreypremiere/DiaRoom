package main

import (
	apperrors "account-microservice/app-errors"
	"account-microservice/contracts/account/requests"
	"account-microservice/contracts/account/responses"
	"account-microservice/database"
	"account-microservice/repositories"
	"account-microservice/services"
	"account-microservice/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

func (a *App) sendError(w http.ResponseWriter, err error) {
    var appErr apperrors.AppError
    
    // По умолчанию считаем ошибку внутренней (500)
    status := http.StatusInternalServerError
    
    if errors.As(err, &appErr) {
        switch appErr.Code {
        case "NOT_FOUND":
            status = http.StatusNotFound
        case "ALREADY_EXISTS":
            status = http.StatusConflict
        case "METHOD_NOT_ALLOWED":
            status = http.StatusMethodNotAllowed 
        case "UNSUPPORTED_TYPE":
            status = http.StatusUnsupportedMediaType 
        default:
            status = http.StatusBadRequest
        }
    } else {
        appErr = apperrors.ErrInternal
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error_code": appErr.Code,
        "message":    appErr.Message,
    })
}

func (a *App) newAccount(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    var newAccount requests.CreatingAccount
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields() 
    
    if err := decoder.Decode(&newAccount); err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    newId, err := a.accountService.NewAccount(r.Context(), &newAccount)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated) 
    json.NewEncoder(w).Encode(map[string]uuid.UUID{"userId": *newId})
}

func (a *App) verify(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    userIDStr := r.PathValue("userId")
    if userIDStr == "" {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    var userVerify requests.VerifyUser

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&userVerify); err != nil {
		a.sendError(w, err)
		return
	}

    userVerify.UserId = userID

    response, err := a.accountService.VerifyCode(r.Context(), &userVerify)

    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) LoginUser(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

	var loginUser requests.LoginUser

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&loginUser); err != nil {
		a.sendError(w, apperrors.ErrInternal)
		return
	}

	user, err := a.accountService.LoginUser(r.Context(), &loginUser)
	if err != nil {
		a.sendError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(responses.LoginResponse{UserId: user.ID, Email: user.Email})
}

func (a *App) repeatCode(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

	userIDStr := r.PathValue("userId")
    if userIDStr == "" {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
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
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    var request struct {
        RefreshToken string `json:"refreshToken"`
    }

    decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() 

	if err := decoder.Decode(&request); err != nil {
		a.sendError(w, apperrors.ErrInternal)
		return
	}

	// Если токен истек или его нет
    response, err := a.accountService.RefreshSession(r.Context(), request.RefreshToken)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    var request struct {
        RefreshToken string `json:"refreshToken"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    err := a.accountService.Logout(r.Context(), request.RefreshToken)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (a *App) getRoom(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

	roomIDStr := r.PathValue("roomId")
    if roomIDStr == "" {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    roomId, err := uuid.Parse(roomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    room, err := a.accountService.GetRoom(r.Context(), roomId)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(room)
}

func (a *App) updateRoom(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    roomID := r.Header.Get("X-Room-ID")

    roomId, err := uuid.Parse(roomID)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    var request requests.UpdateRoomRequest 
    decoder := json.NewDecoder(r.Body)
    decoder.DisallowUnknownFields()

    if err := decoder.Decode(&request); err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    response, err := a.accountService.UpdateRoom(r.Context(), roomId, &request)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

func (a *App) getRoomsInfo(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    var req requests.GetRoomsBatch
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    roomsMap, err := a.accountService.GetRoomsInfoBatch(r.Context(), req.UserIDs)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(roomsMap)
}

func (a *App) getRoomInfoById(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    roomIDStr := r.PathValue("roomId")
    roomId, err := uuid.Parse(roomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    room, err := a.accountService.GetRoomInfo(r.Context(), roomId)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(room)
}

func (a *App) getRoomInfo(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    if r.Header.Get("Content-Type") != "application/json" {
        a.sendError(w, apperrors.ErrUnsupportedType)
        return
    }

    var req struct {
        RoomId uuid.UUID `json:"room_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    room, err := a.accountService.GetRoomInfo(r.Context(), req.RoomId)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(room)
}

func (a *App) checkSubscription(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    // Извлекаем ID комнаты из пути /account/checkRoomSubscription/{roomId}
    targetRoomIDStr := r.PathValue("roomId")
    targetRoomId, err := uuid.Parse(targetRoomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    // Получаем ID комнаты текущего пользователя (кто залогинен)
    myRoomIDStr := r.Header.Get("X-Room-ID")
    myRoomId, err := uuid.Parse(myRoomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    isFollowed, err := a.accountService.CheckSubscription(r.Context(), myRoomId, targetRoomId)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "result": isFollowed,
    })
}

// followRoom создает запись в таблице подписок
func (a *App) followRoom(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    myRoomIDStr := r.Header.Get("X-Room-ID")
    myRoomId, err := uuid.Parse(myRoomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    var req struct {
        FollowingID uuid.UUID `json:"following_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    if myRoomId == req.FollowingID {
        a.sendError(w, apperrors.ErrInvalidInput) 
        return
    }

    err = a.accountService.Follow(r.Context(), myRoomId, req.FollowingID)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.WriteHeader(http.StatusCreated)
}

func (a *App) unfollowRoom(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        a.sendError(w, apperrors.ErrMethodNotAllowed)
        return
    }

    myRoomIDStr := r.Header.Get("X-Room-ID")
    myRoomId, err := uuid.Parse(myRoomIDStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInternal)
        return
    }

    var req struct {
        FollowingID uuid.UUID `json:"following_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        a.sendError(w, apperrors.ErrInvalidInput)
        return
    }

    err = a.accountService.Unfollow(r.Context(), myRoomId, req.FollowingID)
    if err != nil {
        a.sendError(w, err)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (a *App) GetFollowersHandler(w http.ResponseWriter, r *http.Request) {
	roomIdStr := r.PathValue("roomId") 

    if roomIdStr == "" {
        a.sendError(w, apperrors.ErrInvalidInput) 
        return
    }

    roomId, err := uuid.Parse(roomIdStr)
    if err != nil {
        a.sendError(w, apperrors.ErrInvalidInput) 
        return
    }

    query := r.URL.Query()

    page, err := strconv.Atoi(query.Get("page"))
    if err != nil || page < 1 {
        page = 1
        fmt.Println("Ошибка извлечения параметров page")
    }

    limit, err := strconv.Atoi(query.Get("limit"))
    if err != nil || limit < 1 || limit > 100 { 
        limit = 20
        fmt.Println("Ошибка извлечения параметров limit")
    }


	authors, err := a.accountService.GetRoomFollowers(r.Context(), roomId, page, limit)
	if err != nil {
		a.sendError(w, err)
		return
	}

	response := map[string]interface{}{
		"authors": authors, 
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}


func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

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
        log.Fatalf("Критическая ошибка: не удалось инициализировать пул БД: %v", err)
    }
    defer poolPg.Close() 
    fmt.Println("Пул соединений PostgreSQL инициализирован")

	hostRedis := os.Getenv("REDIS_OTP_HOST")
	portRedis := os.Getenv("REDIS_OTP_PORT")
	addrRedis := fmt.Sprintf("%s:%s", hostRedis, portRedis)
	rdb := redis.NewClient(&redis.Options{Addr: addrRedis})
    defer rdb.Close() 

    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatalf("Критическая ошибка: Redis недоступен: %v", err)
    }

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
    if secretJwt == "" {
        log.Fatal("Критическая ошибка: JWT_SECRET не задан")
    }

	jwtmanager := jwtmanager.NewJWTManager(secretJwt, 15*time.Minute)

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
    mux.HandleFunc("GET /getRoomInfoById/{roomId}", app.getRoomInfoById)
    mux.HandleFunc("GET /checkRoomSubscription/{roomId}", app.checkSubscription)
    mux.HandleFunc("DELETE /unfollowRoom", app.unfollowRoom)
    mux.HandleFunc("POST /followRoom", app.followRoom)
    mux.HandleFunc("GET /followers/{roomId}", app.GetFollowersHandler)

    //Внутренние
    mux.HandleFunc("POST /getRoomsInfoInternal", app.getRoomsInfo)
    mux.HandleFunc("POST /getRoomInfoInternal", app.getRoomInfo)

	server := &http.Server{
        Addr:    ":81",
        Handler: mux,
    }

    // Запуск сервера в отдельной горутине
    go func() {
        fmt.Println("Сервер запущен на :81")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Ошибка при работе сервера: %v", err)
        }
    }()

    <-ctx.Done()
    fmt.Println("\nПолучен сигнал завершения, останавливаем сервер...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        fmt.Printf("Ошибка при плавной остановке: %v\n", err)
    }

    fmt.Println("Сервер успешно остановлен")
}