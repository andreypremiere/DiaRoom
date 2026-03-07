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

type ProxyHandler struct {
	targets map[string]*httputil.ReverseProxy
	protectedPaths map[string][]string
	jwtManager *jwtmanager.JWTManager// Добавить класс
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		targets:        make(map[string]*httputil.ReverseProxy),
		protectedPaths: make(map[string][]string), // ОБЯЗАТЕЛЬНО инициализировать
		jwtManager:     jwtmanager.NewJWTManager(os.Getenv("JWT_SECRET"), 30*time.Minute),
	}
}

func (p *ProxyHandler) AddRoute(prefix string, targetAddress string, protected []string) {
	target, _ := url.Parse(targetAddress)
	proxy := httputil.NewSingleHostReverseProxy(target)

	// originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
        r.URL.Host = target.Host
		// originalDirector(r)
		// r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		// r.Header.Set("X-Gateway-Auth", "trusted")
	}

	p.targets[prefix] = proxy
    p.protectedPaths[prefix] = protected
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for prefix, proxy := range p.targets {
		if strings.HasPrefix(r.URL.Path, prefix) {


			// Проверяем, нужно ли требовать JWT для этого конкретного пути
            needsAuth := false
            for _, path := range p.protectedPaths[prefix] {
                // Если путь в запросе совпадает с защищенным (напр. "/rooms/create")
                if r.URL.Path == path {
                    needsAuth = true
                    break
                }
            }

            if needsAuth {
				// 1. Достаем заголовок
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					http.Error(w, "Отсутствует токен", http.StatusUnauthorized)
					return
				}

				// 2. Очищаем от приставки "Bearer "
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")

                claims, error := p.jwtManager.Verify(tokenString)
                if error != nil {
                    http.Error(w, "Доступ запрещен", http.StatusUnauthorized)
                    return
                }
                r.Header.Set("X-User-ID", claims.UserId)
				r.Header.Set("X-Room-ID", claims.RoomId)
            }

            // Теперь обрезаем префикс перед отправкой в микросервис
            r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
            proxy.ServeHTTP(w, r)
            return
		}
	}

	http.Error(w, "Сервис не найден", http.StatusNotFound)
}

func main() {
	gateway := NewProxyHandler()

	gateway.AddRoute("/auth", "http://user-microservice:81", []string{})
	gateway.AddRoute("/rooms", "http://room-microservice:81", []string{"/rooms/getRoomByRoomId"})
	http.ListenAndServe(":80", gateway)
}