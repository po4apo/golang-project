package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	
	"golang-project/services/auth-service/internal/domain"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, email, passHash string) (string, error) {
	userID := uuid.New().String()

	query := `
		INSERT INTO users (id, email, pass_hash, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	
	_, err := r.db.ExecContext(ctx, query, userID, email, passHash)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (r *UserRepo) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	query := `
		SELECT id, email, pass_hash, created_at
		FROM users
		WHERE email = $1
	`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PassHash,
		&user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	var user domain.User
	
	query := `
		SELECT id, email, pass_hash, created_at
		FROM users
		WHERE id = $1
	`
	
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PassHash,
		&user.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}
