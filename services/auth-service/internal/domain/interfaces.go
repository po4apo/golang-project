package domain

import "context"

// UserRepository — интерфейс для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, email, passHash string) (string, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, userID string) (*User, error)
}

// PasswordHasher — интерфейс для хеширования паролей
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// TokenGenerator — интерфейс для работы с токенами
type TokenGenerator interface {
	Generate(userID string) (accessToken, refreshToken string, err error)
	Validate(token string) (userID string, err error)
}

// User — доменная модель пользователя
type User struct {
	ID        string
	Email     string
	PassHash  string
	CreatedAt string
}

