package main

import (
	"context"
	"diary-microservice/database"
	"diary-microservice/repositories"
	"diary-microservice/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type App struct {
	service *services.DiaryService
}

func (a *App) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
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