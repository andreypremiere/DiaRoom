package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp" // Добавили для работы с динамическими путями
	"strings"
	"time"

	"github.com/andreypremiere/jwtmanager"
)

type ProxyHandler struct {
	targets        map[string]*httputil.ReverseProxy
	protectedPaths map[string][]*regexp.Regexp // Теперь здесь скомпилированные регулярки
	jwtManager     *jwtmanager.JWTManager
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		targets:        make(map[string]*httputil.ReverseProxy),
		protectedPaths: make(map[string][]*regexp.Regexp),
		jwtManager:     jwtmanager.NewJWTManager(os.Getenv("JWT_SECRET"), 30*time.Minute),
	}
}

func (p *ProxyHandler) AddRoute(prefix string, targetAddress string, protected []string) {
	target, _ := url.Parse(targetAddress)
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		// Важно: Host заголовок тоже нужно обновлять для многих серверов
		r.Host = target.Host 
	}

	p.targets[prefix] = proxy

	// Компилируем строки путей в регулярные выражения один раз при запуске
	var regexes []*regexp.Regexp
	for _, pathStr := range protected {
		// Создаем регулярку. ^ и $ гарантируют полное совпадение строки
		expr, err := regexp.Compile("^" + pathStr + "$")
		if err == nil {
			regexes = append(regexes, expr)
		}
	}
	p.protectedPaths[prefix] = regexes
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for prefix, proxy := range p.targets {
		if strings.HasPrefix(r.URL.Path, prefix) {
			
			// Сохраняем оригинальный путь для проверки авторизации
			originalPath := r.URL.Path

			needsAuth := false
			for _, re := range p.protectedPaths[prefix] {
				if re.MatchString(originalPath) {
					needsAuth = true
					break
				}
			}

			if needsAuth {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
					http.Error(w, "Unauthorized: Missing Token", http.StatusUnauthorized)
					return
				}

				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := p.jwtManager.Verify(tokenString)
				if err != nil {
					http.Error(w, "Unauthorized: Invalid Token", http.StatusUnauthorized)
					return
				}
				
				r.Header.Set("X-User-ID", claims.UserId)
				r.Header.Set("X-Room-ID", claims.RoomId)
			}

			// Удаляем префикс (например "/post") перед проксированием
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			proxy.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, "Service Not Found", http.StatusNotFound)
}

func main() {
    gateway := NewProxyHandler()

    // Регистрация путей микросервиса постов
    gateway.AddRoute("/post", "http://post-microservice:81", []string{
        "/post/getPresignedUrls",
        "/post/createPost",
        "/post/publishPost",
        // Маска для динамического ID: /post/ЛЮБОЙ_ID/canvas
        "/post/[a-zA-Z0-9-]+/canvas", 
    })

    // Для комнат
    gateway.AddRoute("/rooms", "http://room-microservice:81", []string{
        "/rooms/getRoomByRoomId",
    })

    http.ListenAndServe(":80", gateway)
}