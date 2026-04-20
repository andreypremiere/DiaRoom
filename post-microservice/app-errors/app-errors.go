package apperrors

type AppError struct {
	Code string
	Message string
}

func (a AppError) Error() string {
	return a.Message
}

var (
	ErrNotFound = AppError{Code: "NOT_FOUND", Message: "Объект не найден"}
	ErrAlreadyExists = AppError{Code: "ALREADY_EXISTS", Message: "Объект уже существует"}
    ErrInvalidInput = AppError{Code: "INVALID_INPUT", Message: "Некорректные входные данные"}
    ErrInternal = AppError{Code: "INTERNAL_ERROR", Message: "Внутренняя ошибка сервера"}
	ErrInvalidPassword = AppError{Code: "INVALID_PASSWORD", Message: "Неверный пароль"}
	ErrInvalidCode = AppError{Code: "INVALID_VERIFICATION_CODE", Message: "Неверный проверочный код"}
	ErrCodeExpired = AppError{Code: "CODE_EXPIRED", Message: "Срок действия кода истек"}
	ErrGeneratingLinksForMedia = AppError{Code: "MEDIA_LINKS_FAILED", Message: "Ошибка генерации ссылок для медиа"}
	ErrSessionExpired = AppError{Code: "SESSION_LIFE_EXPIRED", Message: "Срок действия сессии истек"}
	ErrEmailProvider = AppError{Code: "EMAIL_PROVIDER_FAILED", Message: "Ошибка в работе провайдера по отправке почтовых писем"}
	ErrMethodNotAllowed = AppError{Code: "METHOD_NOT_ALLOWED", Message: "Данный метод не поддерживается"}
	ErrUnsupportedType = AppError{Code: "UNSUPPORTED_TYPE", Message: "Данный метод не поддерижвает такой тип данных"}
)

