package config

import (
	"github.com/spf13/viper"
)

type Config struct {
    GRPCAddr     string
    HTTPAddr     string
    AuthGRPCAddr string
    DBDSN        string
    JWTKey       string
    LogLevel     string
}

func Load() *Config {
    // Дефолтные значения
	viper.SetDefault("grpc_addr", ":50051")
	viper.SetDefault("http_addr", ":8080")
	viper.SetDefault("auth_grpc_addr", "localhost:50051")
	viper.SetDefault("db_dsn", "postgres://authuser:authpass@localhost:5432/authdb?sslmode=disable")
	viper.SetDefault("jwt_key", "secret")
    viper.SetDefault("log_level", "info")
    
    // Читать из env переменных
    viper.AutomaticEnv()
    
    return &Config{
        GRPCAddr:     viper.GetString("grpc_addr"),
        HTTPAddr:     viper.GetString("http_addr"),
        AuthGRPCAddr: viper.GetString("auth_grpc_addr"),
        DBDSN:        viper.GetString("db_dsn"),
        JWTKey:       viper.GetString("jwt_key"),
        LogLevel:     viper.GetString("log_level"),
    }
}