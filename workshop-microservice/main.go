package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	apperrors "workshop-microservice/app-errors"
	"workshop-microservice/contracts/requests"
	"workshop-microservice/database"
	"workshop-microservice/repositories"
	"workshop-microservice/services"

	"github.com/google/uuid"
)

type App struct {
	service *services.WorkshopService
}

func (a *App) sendError(w http.ResponseWriter, err error) {
	var appErr apperrors.AppError

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
		case "INVALID_INPUT":
			status = http.StatusBadRequest
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

func (a *App) handleRenameFolder(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	folderIDStr := r.PathValue("folderId")
	if folderIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	var folder struct {
		FolderName string `json:"folderName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&folder); err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}
	defer r.Body.Close()

	err = a.service.RenameFolder(r.Context(), roomID, folderID, folder.FolderName)
	if err != nil {
		a.sendError(w, err)
		return 
	}

	w.WriteHeader(http.StatusOK)
}

func (a *App) handleGetRoot(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	_, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	roomIDPathStr := r.PathValue("roomId")
	if roomIDPathStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	roomIDPath, err := uuid.Parse(roomIDPathStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	resultRoot, err := a.service.GetRoot(r.Context(), roomIDPath)
	if err != nil {
		a.sendError(w, err)
		return 
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resultRoot)
}

func (a *App) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	var newFolder requests.CreateFolder
	if err := json.NewDecoder(r.Body).Decode(&newFolder); err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}
	defer r.Body.Close()

	newFolder.RoomId = &roomID

	folder, err := a.service.CreateFolder(r.Context(), &newFolder)
	if err != nil {
		a.sendError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func (a *App) handleMoveFolder(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	var moving requests.MoveFolder
	if err := json.NewDecoder(r.Body).Decode(&moving); err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}
	defer r.Body.Close()

	err = a.service.MoveFolder(r.Context(), roomID, &moving)
	if err != nil {
		a.sendError(w, err)
	}

	w.WriteHeader(http.StatusOK)
}

func (a *App) handleGetFolder(w http.ResponseWriter, r *http.Request) {
	roomIDStr := r.Header.Get("X-Room-ID")
	if roomIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	_, err := uuid.Parse(roomIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	folderIDStr := r.PathValue("folderId")
	if folderIDStr == "" {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		a.sendError(w, apperrors.ErrInvalidInput)
		return
	}

	resultFolder, err := a.service.GetFolder(r.Context(), folderID)
	if err != nil {
		a.sendError(w, err)
		return 
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resultFolder)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

	s3Client, err := database.NewS3Client()
	if err != nil {
		log.Fatalf("Критическая ошибка: не удалось создать клиент S3: %v", err)
	}

	pool, err := database.InitPool(ctx)
	if err != nil {
		log.Fatalf("Критическая ошибка: не удалось инициализировать БД: %v", err)
	}
	defer pool.Close()

	repository := repositories.NewWorkshopRepository(pool)
	service := services.NewWorkshopService(repository, s3Client)

	app := App{service: service}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /getRoot/{roomId}", app.handleGetRoot)
	mux.HandleFunc("POST /createFolder", app.handleCreateFolder)
	mux.HandleFunc("PATCH /renameFolder/{folderId}", app.handleRenameFolder)
	mux.HandleFunc("POST /moveFolder", app.handleMoveFolder)
	mux.HandleFunc("GET /getFolder/{folderId}", app.handleGetFolder)

	server := &http.Server{
		Addr:    ":81",
		Handler: mux,
	}

	go func() {
		fmt.Println("Сервер постов запущен на порту :81")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("Завершение работы...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Ошибка при остановке сервера: %v\n", err)
	}

}