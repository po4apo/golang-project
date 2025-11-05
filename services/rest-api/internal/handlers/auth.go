package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	authv1 "golang-project/api/proto/gen/go/auth/v1"
	"golang-project/services/rest-api/internal/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler обрабатывает HTTP запросы для аутентификации
type AuthHandler struct {
	authClient *client.AuthClient
}

// NewAuthHandler создаёт новый обработчик для аутентификации
func NewAuthHandler(authClient *client.AuthClient) *AuthHandler {
	return &AuthHandler{
		authClient: authClient,
	}
}

// SignUpRequest - тело запроса для регистрации
type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignUpResponse - тело ответа для регистрации
type SignUpResponse struct {
	UserID string `json:"user_id"`
}

// SignInRequest - тело запроса для входа
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignInResponse - тело ответа для входа
type SignInResponse struct {
	Token string `json:"token"`
}

// ValidateTokenResponse - тело ответа для валидации токена
type ValidateTokenResponse struct {
	UserID string `json:"user_id"`
	Valid  bool   `json:"valid"`
}

// ErrorResponse - стандартный ответ об ошибке
type ErrorResponse struct {
	Error string `json:"error"`
}

// SignUp обрабатывает POST /api/v1/auth/signup
func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode request", "error", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authClient.Client.SignUp(context.Background(), &authv1.SignUpRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		handleGRPCError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, SignUpResponse{
		UserID: resp.UserId,
	})
}

// SignIn обрабатывает POST /api/v1/auth/signin
func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode request", "error", err)
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authClient.Client.SignIn(context.Background(), &authv1.SignInRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		handleGRPCError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, SignInResponse{
		Token: resp.AccessToken,
	})
}

// ValidateToken обрабатывает GET /api/v1/auth/validate
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		respondError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	// Убираем "Bearer " prefix если есть
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	resp, err := h.authClient.Client.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{
		Token: token,
	})

	if err != nil {
		handleGRPCError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, ValidateTokenResponse{
		UserID: resp.UserId,
		Valid:  resp.Valid,
	})
}

// respondJSON отправляет JSON ответ
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// respondError отправляет JSON ответ с ошибкой
func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, ErrorResponse{Error: message})
}

// handleGRPCError конвертирует gRPC ошибку в HTTP статус
func handleGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var httpStatus int
	switch st.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	default:
		httpStatus = http.StatusInternalServerError
	}

	respondError(w, httpStatus, st.Message())
}

