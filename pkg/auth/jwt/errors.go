package jwt

import "errors"

// Предопределённые ошибки для JWT операций
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrExpiredToken         = errors.New("token expired")
	ErrInvalidSigningMethod = errors.New("invalid signing method")
	ErrMissingKey           = errors.New("missing key")
)

