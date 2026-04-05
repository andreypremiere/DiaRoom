package requests

type UserCreatingContract struct {
	Email string `json:"email"`
	Password string `json:"password"` 
}