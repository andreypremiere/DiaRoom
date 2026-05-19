package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Встраиваем папку templates в бинарный файл
//go:embed templates/*
var templateFS embed.FS

// TemplateRegistry для рендеринга HTML в Echo
type TemplateRegistry struct {
	templates *template.Template
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()

	// Middleware для логирования и восстановления после паник
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Инициализация шаблонизатора
	t := &TemplateRegistry{
		templates: template.Must(template.ParseFS(templateFS, "templates/*.html")),
	}
	e.Renderer = t

	// 1. Главная страница
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})

	e.GET("/about", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "about us")
	})

	e.GET("/favicon.ico", func(c echo.Context) error {
    file, err := templateFS.ReadFile("templates/favicon.ico")
    if err != nil {
        return c.NoContent(http.StatusNoContent)
    }
    return c.Blob(http.StatusOK, "image/x-icon", file)
})

	// 2. Роут для скачивания приложения (редирект на S3)
	e.GET("/download/:os", func(c echo.Context) error {
		osType := c.Param("os")
		
		// Здесь указываешь свои реальные ссылки на S3 Bucket
		s3BaseURL := "https://your-s3-bucket.s3.eu-central-1.amazonaws.com/releases/"
		
		switch osType {
		case "android":
			// Прямая ссылка на APK файл
			return c.Redirect(http.StatusTemporaryRedirect, s3BaseURL+"diaroom-latest.apk")
		case "ios":
			// Для iOS обычно используется TestFlight или App Store
			// Но если файл лежит в S3 (например, ipa для Enterprise), отдаем его
			return c.Redirect(http.StatusTemporaryRedirect, s3BaseURL+"diaroom-latest.ipa")
		default:
			return c.String(http.StatusBadRequest, "Неизвестная операционная система")
		}
	})

	// Запуск на порту 81 (для API Gateway)
	port := os.Getenv("PORT")
	if port == "" {
		port = "81"
	}
	e.Logger.Fatal(e.Start(":" + port))
}