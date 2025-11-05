package main

import (
	"database/sql"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	authv1 "golang-project/api/proto/gen/go/auth/v1"
	"golang-project/pkg/config"
	"golang-project/services/auth-service/internal/hash"
	"golang-project/services/auth-service/internal/repo"
	"golang-project/services/auth-service/internal/service"
)

func main() {
	cfg := config.Load()
	
	// Подключение к БД
	db, err := sql.Open("postgres", cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	
	// Инициализация зависимостей
	userRepo := repo.NewUserRepo(db)
	hasher := hash.NewArgon2Hasher()
	authService := service.NewAuthServer(userRepo, hasher)
	
	// Запуск gRPC сервера
	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, authService)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("auth gRPC listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	grpcServer.GracefulStop()
}