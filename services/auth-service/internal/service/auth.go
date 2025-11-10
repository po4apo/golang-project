package service

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "golang-project/api/proto/gen/go/auth/v1"
	"golang-project/pkg/auth/jwt"
	"golang-project/services/auth-service/internal/hash"
	"golang-project/services/auth-service/internal/repo"
	"golang-project/services/auth-service/internal/validator"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	repo   *repo.UserRepo
	hasher *hash.Argon2Hasher
	jwt    *jwt.Manager
}

func NewAuthServer(userRepo *repo.UserRepo, hasher *hash.Argon2Hasher, jwtManager *jwt.Manager) *AuthServer {
	slog.Info("creating auth service")
	return &AuthServer{
		repo:   userRepo,
		hasher: hasher,
		jwt:    jwtManager,
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
	
	// Генерация JWT токена
	accessToken, err := s.jwt.Sign(user.ID)
	if err != nil {
		slog.Error("failed to generate access token", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}
	
	// TODO: Refresh token будет реализован позже
	refreshToken := ""
	
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
	
	// Валидация JWT токена
	claims, err := s.jwt.Validate(req.Token)
	if err != nil {
		if err == jwt.ErrExpiredToken {
			slog.Warn("token expired", slog.String("op", op))
			return &authv1.ValidateTokenResponse{
				UserId: "",
				Valid:  false,
			}, nil
		}
		if err == jwt.ErrInvalidToken {
			slog.Warn("invalid token", slog.String("op", op), slog.Any("error", err))
			return &authv1.ValidateTokenResponse{
				UserId: "",
				Valid:  false,
			}, nil
		}
		slog.Error("failed to validate token", slog.String("op", op), slog.Any("error", err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	
	slog.Info("token validated", slog.String("op", op), slog.String("user_id", claims.UserID))
	
	return &authv1.ValidateTokenResponse{
		UserId: claims.UserID,
		Valid:  true,
	}, nil
}
