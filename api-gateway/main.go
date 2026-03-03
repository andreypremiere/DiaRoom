package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type ProxyHandler struct {
	targets map[string]*httputil.ReverseProxy
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{targets: make(map[string]*httputil.ReverseProxy)}
}

func (p *ProxyHandler) AddRoute(prefix string, targetAddress string) {
	target, _ := url.Parse(targetAddress)
	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		r.Header.Set("X-Gateway-Auth", "trusted")
	}

	p.targets[prefix] = proxy
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for prefix, proxy := range p.targets {
		if strings.HasPrefix(r.URL.Path, prefix) {
			log.Printf("Маршрутизация: %s -> %s: ", r.URL.Path, prefix)
			proxy.ServeHTTP(w, r)
			return 
		}
	}

	http.Error(w, "Сервис не найден", http.StatusNotFound)
}

func main() {
	gateway := NewProxyHandler()

	gateway.AddRoute("/auth", "http://user-microservice:81")
	gateway.AddRoute("/rooms", "http://room-microservice:81")
	http.ListenAndServe(":80", gateway)
}