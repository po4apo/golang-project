package service

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "golang-project/api/proto/gen/go/auth/v1"
	"golang-project/services/auth-service/internal/hash"
	"golang-project/services/auth-service/internal/repo"
	"golang-project/services/auth-service/internal/validator"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	repo   *repo.UserRepo
	hasher *hash.Argon2Hasher
}

func NewAuthServer(userRepo *repo.UserRepo, hasher *hash.Argon2Hasher) *AuthServer {
	slog.Info("creating auth service")
	return &AuthServer{
		repo:   userRepo,
		hasher: hasher,
	}
}

// SignUp регистрирует нового пользователя
func (s *AuthServer) SignUp(ctx context.Context, req *authv1.SignUpRequest) (*authv1.SignUpResponse, error) {
	op := "SignUp"
	
	slog.Info("sign up attempt", slog.String("op", op), slog.String("email", req.Email))
	
	// Валидация
	if err := validator.ValidateEmail(req.Email); err != nil {
		slog.Warn("invalid email", slog.String("op", op), slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	
	if err := validator.ValidatePassword(req.Password); err != nil {
		slog.Warn("invalid password", slog.String("op", op), slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	
	// Проверка существования
	exists, err := s.repo.UserExistsByEmail(ctx, req.Email)
	if err != nil {
		slog.Error("failed to check user existence", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	if exists {
		slog.Warn("user already exists", slog.String("op", op), slog.String("email", req.Email))
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}
	
	// Хеширование пароля
	passHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		slog.Error("failed to hash password", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	
	// Создание пользователя
	userID, err := s.repo.CreateUser(ctx, req.Email, passHash)
	if err != nil {
		slog.Error("failed to create user", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	
	slog.Info("user created", slog.String("op", op), slog.String("user_id", userID), slog.String("email", req.Email))
	
	return &authv1.SignUpResponse{UserId: userID}, nil
}

// SignIn аутентифицирует пользователя
func (s *AuthServer) SignIn(ctx context.Context, req *authv1.SignInRequest) (*authv1.SignInResponse, error) {
	op := "SignIn"
	
	slog.Info("sign in attempt", slog.String("op", op), slog.String("email", req.Email))
	
	// Валидация
	if err := validator.ValidateEmail(req.Email); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password required")
	}
	
	// Получить пользователя
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == repo.ErrUserNotFound {
		slog.Warn("user not found", slog.String("op", op), slog.String("email", req.Email))
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		slog.Error("failed to get user", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	
	// Проверить пароль
	valid, err := s.hasher.Verify(req.Password, user.PassHash)
	if err != nil {
		slog.Error("failed to verify password", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !valid {
		slog.Warn("invalid password", slog.String("op", op), slog.String("email", req.Email))
		return nil, status.Error(codes.Unauthenticated, "invalid password")
	}
	
	// TODO: Генерация JWT токена (следующий пункт)
	accessToken := "temporary_token"
	refreshToken := "temporary_refresh"
	
	slog.Info("user signed in", slog.String("op", op), slog.String("user_id", user.ID))
	
	return &authv1.SignInResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateToken проверяет токен
func (s *AuthServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	op := "ValidateToken"
	
	slog.Info("validate token", slog.String("op", op))
	
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token required")
	}
	
	// TODO: Проверка JWT токена (следующий пункт)
	
	return &authv1.ValidateTokenResponse{
		UserId: "123",
		Valid:  true,
	}, nil
}
