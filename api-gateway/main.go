package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/andreypremiere/jwtmanager"
)

// Обертка для перехвата статус-кода ответа
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

type ProxyHandler struct {
	targets        map[string]*httputil.ReverseProxy
	protectedPaths map[string][]*regexp.Regexp
	jwtManager     *jwtmanager.JWTManager
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		targets:        make(map[string]*httputil.ReverseProxy),
		protectedPaths: make(map[string][]*regexp.Regexp),
		jwtManager:     jwtmanager.NewJWTManager(os.Getenv("JWT_SECRET"), 15*time.Minute),
	}
}

func (p *ProxyHandler) AddRoute(prefix string, targetAddress string, protected []string) {
	target, _ := url.Parse(targetAddress)
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.Host = target.Host
	}

	p.targets[prefix] = proxy

	var regexes []*regexp.Regexp
	for _, pathStr := range protected {
		expr, err := regexp.Compile("^" + pathStr + "$")
		if err == nil {
			regexes = append(regexes, expr)
		}
	}
	p.protectedPaths[prefix] = regexes
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Оборачиваем ResponseWriter
	lrw := &loggingResponseWriter{w, http.StatusOK}

	defer func() {
		// Логируем после завершения обработки
		duration := time.Since(start)
		log.Printf(
			"[%s] %d | %-7s | %s | %s",
			time.Now().Format("15:04:05"),
			lrw.statusCode,
			r.Method,
			r.URL.Path,
			duration,
		)
	}()

	var targetPrefix string
	var proxy *httputil.ReverseProxy

	for prefix, prx := range p.targets {
		if strings.HasPrefix(r.URL.Path, prefix) {
			targetPrefix = prefix
			proxy = prx
			break
		}
	}

	if proxy == nil {
		http.Error(lrw, "Service Not Found", http.StatusNotFound)
		return
	}

	originalPath := r.URL.Path

	needsAuth := false
	for _, re := range p.protectedPaths[targetPrefix] {
		if re.MatchString(originalPath) {
			needsAuth = true
			break
		}
	}

	if needsAuth {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(lrw, "Unauthorized: Missing Token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := p.jwtManager.Verify(tokenString)
		if err != nil {
			http.Error(lrw, "Unauthorized: Invalid or Expired Token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-User-ID", claims.UserId)
		r.Header.Set("X-Room-ID", claims.RoomId)
	}

	r.URL.Path = strings.TrimPrefix(r.URL.Path, targetPrefix)
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	proxy.ServeHTTP(lrw, r)
}

func main() {
	gateway := NewProxyHandler()

	fmt.Println("API Gateway started on :80")

	gateway.AddRoute("/post", "http://post-microservice:81", []string{
		"/post/getPresignedUrls",
		"/post/createPost",
		"/post/publishPost",
		"/post/saveCanvas/[a-zA-Z0-9-]+",
		"/post/getPersonalPosts",
	})

	gateway.AddRoute("/rooms", "http://room-microservice:81", []string{
		"/rooms/getRoomByRoomId",
	})

	gateway.AddRoute("/account", "http://account-microservice:81", []string{
		"/account/updateRoom", "/account/room/[a-zA-Z0-9-]+",
	})

	log.Fatal(http.ListenAndServe(":80", gateway))
}