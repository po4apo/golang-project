package validator

import (
	"errors"
	"regexp"
)

var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordTooShort = errors.New("password too short (min 8 chars)")
	ErrPasswordTooWeak  = errors.New("password too weak")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail проверяет формат email
func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword проверяет требования к паролю
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	// Можно добавить проверку на сложность (цифры, спецсимволы)
	return nil
}