package main

import (
	"context"
	apperrors "diary-microservice/app-errors"
	"diary-microservice/contracts/requests"
	"diary-microservice/database"
	"diary-microservice/repositories"
	"diary-microservice/services"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type App struct {
	service *services.DiaryService
}

func (a *App) sendError(ctx echo.Context, err error) error {
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

	return ctx.JSON(status, map[string]string{
		"error_code": appErr.Code,
		"message":    appErr.Message,
	})
}

func (a *App) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (a *App) createMessage(c echo.Context) error {
	roomIdStr := c.Request().Header.Get("X-Room-ID")
	if roomIdStr == "" {
		return a.sendError(c, apperrors.ErrAccess)
	}
	roomId, err := uuid.Parse(roomIdStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	req := new(requests.MessageCreateRequest)
    
    if err := c.Bind(req); err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput) 
    }
	fmt.Println("Это запрос:")
	spew.Dump(req)

	ctx := c.Request().Context()
	response, err := a.service.CreateMessage(ctx, roomId, req)

	fmt.Println("Это ответ:")
	spew.Dump(response)

	return c.JSON(http.StatusCreated, response)
}

func (a *App) updateStatusMessage(c echo.Context) error {
	roomIdStr := c.Request().Header.Get("X-Room-ID")
	if roomIdStr == "" {
		return a.sendError(c, apperrors.ErrAccess)
	}
	roomId, err := uuid.Parse(roomIdStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	req := new(requests.UpdatingMessage)
    
    if err := c.Bind(req); err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput) 
    }

	ctx := c.Request().Context()
	response, err := a.service.UpdateMessageStatus(ctx, roomId, req)
	if err != nil {
		return a.sendError(c, err)
	}

	return c.JSON(http.StatusOK, response)
}

func (a *App) GetMessages(c echo.Context) error {
	roomIDStr := c.Param("roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 0 {
		page = 0
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Рассчитываем смещение (offset)
	offset := page * limit

	ctx := c.Request().Context()
	messages, err := a.service.GetMessages(ctx, roomID, limit, offset)
	if err != nil {
		return a.sendError(c, err)
	}

	return c.JSON(http.StatusOK, messages)
}

func (a *App) CreateTag(c echo.Context) error {
	roomIdStr := c.Request().Header.Get("X-Room-ID")
	if roomIdStr == "" {
		return a.sendError(c, apperrors.ErrAccess)
	}
	roomId, err := uuid.Parse(roomIdStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	req := &requests.CreatingTag{}
    
    if err := c.Bind(req); err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput) 
    }

	newTag, err := a.service.CreateTag(c.Request().Context(), req, roomId)
	if err != nil {
		return a.sendError(c, err)
	}

	return c.JSON(http.StatusCreated, newTag)
}

func (a *App) UpdateTag(c echo.Context) error {
	roomIdStr := c.Request().Header.Get("X-Room-ID")
	if roomIdStr == "" {
		return a.sendError(c, apperrors.ErrAccess)
	}
	roomId, err := uuid.Parse(roomIdStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	idStr := c.Param("tagId")

    tagId, err := uuid.Parse(idStr)
    if err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput)
    }

	req := &requests.UpdatingTag{}
    
    if err := c.Bind(req); err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput) 
    }

	updatedTag, err := a.service.UpdateTag(c.Request().Context(), req, tagId, roomId)
	if err != nil {
		return a.sendError(c, err)
	}

	return c.JSON(http.StatusCreated, updatedTag)
}

func (a *App) DeleteTag(c echo.Context) error {
    roomIdStr := c.Request().Header.Get("X-Room-ID")
    if roomIdStr == "" {
        return a.sendError(c, apperrors.ErrAccess)
    }
    roomId, err := uuid.Parse(roomIdStr)
    if err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput)
    }

    idStr := c.Param("tagId")
    tagId, err := uuid.Parse(idStr)
    if err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput)
    }

    err = a.service.DeleteTag(c.Request().Context(), tagId, roomId)
    if err != nil {
        return a.sendError(c, err)
    }

    return c.NoContent(http.StatusNoContent)
}

func (a *App) GetTags(c echo.Context) error {
	roomIdStr := c.Request().Header.Get("X-Room-ID")
    if roomIdStr == "" {
        return a.sendError(c, apperrors.ErrAccess)
    }
    roomId, err := uuid.Parse(roomIdStr)
    if err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput)
    }

	roomIdPathStr := c.Param("roomId")
	roomIdPath, err := uuid.Parse(roomIdPathStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	if roomId != roomIdPath {
		return a.sendError(c, apperrors.ErrAccess)
	}

	tags, err := a.service.GetTags(c.Request().Context(), roomId)
	if err != nil {
		return a.sendError(c, err)
	}

	return c.JSON(http.StatusOK, tags)
}

func (a *App) DeleteMessage(c echo.Context) error {
	roomIdStr := c.Request().Header.Get("X-Room-ID")
	if roomIdStr == "" {
		return a.sendError(c, apperrors.ErrAccess)
	}
	roomId, err := uuid.Parse(roomIdStr)
	if err != nil {
		return a.sendError(c, apperrors.ErrInvalidInput)
	}

	messageIdStr := c.Param("messageId")

    messageId, err := uuid.Parse(messageIdStr)
    if err != nil {
        return a.sendError(c, apperrors.ErrInvalidInput)
    }

	err = a.service.DeleteMessage(c.Request().Context(), roomId, messageId)
	if err != nil {
		return a.sendError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}


func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Инициализация PostgreSQL
	dbPool, err := database.NewPostgresPool(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()
	log.Println("PostgreSQL connected")

	// Инициализация Redis
	redisClient, err := database.NewRedisClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis connected")

	// Инициализация s3 клиентов
	s3Client, s3PresignedClient, err := database.NewS3Client(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize s3 client: %v", err)
	}

	// Инициализация S3 Manager
	s3Manager := services.NewS3Manager(s3Client, s3PresignedClient)


	// Сборка слоев 
	repo := repositories.NewDiaryRepository(dbPool)
	service := services.NewDiaryService(repo, s3Manager, redisClient)

	app := App{service: service}

	// Инициализация Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())  // Логирование всех запросов
	e.Use(middleware.Recover()) // Защита от падения (panic)
	e.Use(middleware.CORS())    // Настройка CORS для работы с фронтендом

	// Базовый роут для проверки работоспособности
	e.GET("/health", app.health)
	e.POST("/createMessage", app.createMessage)
	e.POST("/updateStatusMessage", app.updateStatusMessage)
	e.GET("/messages/:roomId", app.GetMessages)
	e.POST("/tag", app.CreateTag)
	e.PATCH("/tag/:tagId", app.UpdateTag)
	e.DELETE("/tag/:tagId", app.DeleteTag)
	e.GET("/tags/:roomId", app.GetTags)
	e.DELETE("/message/:messageId", app.DeleteMessage)
	// e.GET("messages/searchByTagName/:roomId", app.GetMessagesByTagName)
	// e.GET("/messages/searchByContent/:roomId", app.GetMessagesByContent)

	// TODO: Здесь будут роуты для WebSocket и API



	// Graceful Shutdown (Безопасное завершение работы сервера)
	go func() {
		port := "81"
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// Ждем сигнал от ОС (Ctrl+C, Docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
	log.Println("Server gracefully stopped")
}