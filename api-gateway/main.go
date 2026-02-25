package main

import (
	"fmt"
	"net/http"
)

type User struct {
	id int
	code int
	hash string
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	fmt.Println("Пришел запрос на главную страницу")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"Статус": "Запрос был обработан"}`))
}

func setValueRedis(w http.ResponseWriter, r *http.Request) {

}

func pingUserMicroservice(w http.ResponseWriter, r *http.Request) {
	userServiceURL := "http://user-microservice:81"

	_, err := http.Get(userServiceURL)

	if err == nil {
		fmt.Println("Запрос в микросервис юзера успешно выполнен")
	} else {
		fmt.Println("Ошибка:", err.Error())
	}
}

func main() {
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/pingUserService", pingUserMicroservice)
	http.HandleFunc("/setRedisValue", setValueRedis)
	http.ListenAndServe(":80", nil)
}