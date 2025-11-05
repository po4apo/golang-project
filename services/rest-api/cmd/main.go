package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"golang-project/pkg/config"
	"golang-project/pkg/logger"
	"golang-project/services/rest-api/internal/client"
	"golang-project/services/rest-api/internal/handlers"
	custommw "golang-project/services/rest-api/internal/middleware"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем логгер
	logger.InitLogger(cfg.LogLevel)
	slog.Info("REST API starting", "port", cfg.HTTPAddr)

	// Создаём gRPC клиенты
	authClient, err := client.NewAuthClient(cfg.AuthGRPCAddr)
	if err != nil {
		slog.Error("failed to create auth client", "error", err)
		os.Exit(1)
	}
	defer authClient.Close()

	// Создаём роутер
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommw.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Healthcheck
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Handlers
	authHandler := handlers.NewAuthHandler(authClient)

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", authHandler.SignUp)
			r.Post("/signin", authHandler.SignIn)
			r.Get("/validate", authHandler.ValidateToken)
		})
	})

	// HTTP сервер
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("REST API started", "addr", cfg.HTTPAddr)

	// Ожидаем сигнал остановки
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}




