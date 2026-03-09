package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/andreypremiere/jwtmanager"
)

// ProxyHandler управляет маршрутизацией и защитой путей через Reverse Proxy
type ProxyHandler struct {
	targets map[string]*httputil.ReverseProxy
	protectedPaths map[string][]string
	jwtManager *jwtmanager.JWTManager
}

// NewProxyHandler инициализирует обработчик с JWT-менеджером и картами путей
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		targets:        make(map[string]*httputil.ReverseProxy),
		protectedPaths: make(map[string][]string),
		jwtManager:     jwtmanager.NewJWTManager(os.Getenv("JWT_SECRET"), 30*time.Minute),
	}
}

// AddRoute регистрирует новый микросервис и список защищенных путей
func (p *ProxyHandler) AddRoute(prefix string, targetAddress string, protected []string) {
	target, _ := url.Parse(targetAddress)
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Настройка модификации заголовков перед отправкой запроса
	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
        r.URL.Host = target.Host
	}

	p.targets[prefix] = proxy
    p.protectedPaths[prefix] = protected
}

// ServeHTTP обрабатывает входящие запросы, проверяет JWT и проксирует трафик
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Поиск соответствия префикса пути зарегистрированным таргетам
	for prefix, proxy := range p.targets {
		if strings.HasPrefix(r.URL.Path, prefix) {

			// Проверка необходимости авторизации для текущего пути
            needsAuth := false
            for _, path := range p.protectedPaths[prefix] {
                if r.URL.Path == path {
                    needsAuth = true
                    break
                }
            }

            if needsAuth {
				// Проверка наличия и формата заголовка Authorization
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					http.Error(w, "Отсутствует токен", http.StatusUnauthorized)
					return
				}

				tokenString := strings.TrimPrefix(authHeader, "Bearer ")

				// Валидация JWT токена
                claims, error := p.jwtManager.Verify(tokenString)
                if error != nil {
                    http.Error(w, "Доступ запрещен", http.StatusUnauthorized)
                    return
                }
				// Проброс данных пользователя в заголовки для микросервиса
                r.Header.Set("X-User-ID", claims.UserId)
				r.Header.Set("X-Room-ID", claims.RoomId)
            }

			// Удаление префикса шлюза перед отправкой запроса в сервис
            r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
            proxy.ServeHTTP(w, r)
            return
		}
	}

	// Возврат ошибки если путь не совпал ни с одним префиксом
	http.Error(w, "Сервис не найден", http.StatusNotFound)
}

func main() {
	// Создание и настройка API Gateway
	gateway := NewProxyHandler()

	// Регистрация маршрутов для микросервисов
	gateway.AddRoute("/auth", "http://user-microservice:81", []string{})
	gateway.AddRoute("/rooms", "http://room-microservice:81", []string{"/rooms/getRoomByRoomId"})
	http.ListenAndServe(":80", gateway)
}