package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims представляет claims JWT токена
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Manager управляет JWT токенами с использованием RS256
type Manager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	ttl        time.Duration
}

// Config конфигурация для JWT Manager
type Config struct {
	// PrivateKeyPath путь к файлу с приватным ключом (опционально, если используется PrivateKey)
	PrivateKeyPath string
	// PrivateKey содержимое приватного ключа в PEM формате (приоритет над PrivateKeyPath)
	PrivateKey string
	// PublicKeyPath путь к файлу с публичным ключом (опционально, если используется PublicKey)
	PublicKeyPath string
	// PublicKey содержимое публичного ключа в PEM формате (приоритет над PublicKeyPath)
	PublicKey string
	// Issuer issuer токена (обычно название сервиса)
	Issuer string
	// TTL время жизни токена
	TTL time.Duration
}

// NewManager создаёт новый JWT Manager
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Issuer == "" {
		cfg.Issuer = "auth-service"
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour // По умолчанию 24 часа
	}

	// Загрузка приватного ключа
	var privateKey *rsa.PrivateKey
	var err error

	if cfg.PrivateKey != "" {
		privateKey, err = parsePrivateKey([]byte(cfg.PrivateKey))
	} else if cfg.PrivateKeyPath != "" {
		privateKey, err = loadPrivateKeyFromFile(cfg.PrivateKeyPath)
	} else {
		return nil, fmt.Errorf("%w: private key is required", ErrMissingKey)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Загрузка публичного ключа
	var publicKey *rsa.PublicKey

	if cfg.PublicKey != "" {
		publicKey, err = parsePublicKey([]byte(cfg.PublicKey))
	} else if cfg.PublicKeyPath != "" {
		publicKey, err = loadPublicKeyFromFile(cfg.PublicKeyPath)
	} else {
		// Если публичный ключ не указан, извлекаем его из приватного
		publicKey = &privateKey.PublicKey
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &Manager{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     cfg.Issuer,
		ttl:        cfg.TTL,
	}, nil
}

// NewManagerFromEnv создаёт JWT Manager из переменных окружения
func NewManagerFromEnv() (*Manager, error) {
	privateKey := os.Getenv("JWT_RSA_PRIVATE_KEY")
	publicKey := os.Getenv("JWT_RSA_PUBLIC_KEY")
	privateKeyPath := os.Getenv("JWT_RSA_PRIVATE_KEY_PATH")
	publicKeyPath := os.Getenv("JWT_RSA_PUBLIC_KEY_PATH")
	issuer := os.Getenv("JWT_ISSUER")
	ttlStr := os.Getenv("JWT_TTL")

	var ttl time.Duration
	if ttlStr != "" {
		var err error
		ttl, err = time.ParseDuration(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT_TTL format: %w", err)
		}
	}

	return NewManager(Config{
		PrivateKey:     privateKey,
		PublicKey:      publicKey,
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
		Issuer:         issuer,
		TTL:            ttl,
	})
}

// Sign создаёт новый JWT токен для пользователя
func (m *Manager) Sign(userID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// Validate проверяет и парсит JWT токен
func (m *Manager) Validate(tokenString string) (*Claims, error) {
	// Парсинг токена
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверка алгоритма подписи
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("%w: expected RS256, got %v", ErrInvalidSigningMethod, token.Header["alg"])
		}
		return m.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	// Проверка валидности токена
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Извлечение claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: invalid claims type", ErrInvalidToken)
	}

	// Дополнительная проверка issuer
	if claims.Issuer != m.issuer {
		return nil, fmt.Errorf("%w: invalid issuer", ErrInvalidToken)
	}

	return claims, nil
}

// parsePrivateKey парсит приватный RSA ключ из PEM формата
func parsePrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var key *rsa.PrivateKey
	var err error

	// Попытка парсинга PKCS1
	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	// Попытка парсинга PKCS8
	pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	key, ok := pkcs8Key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("key is not RSA private key")
	}

	return key, nil
}

// parsePublicKey парсит публичный RSA ключ из PEM формата
func parsePublicKey(keyData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var key interface{}
	var err error

	// Попытка парсинга PKIX
	key, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Попытка парсинга PKCS1
		key, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("key is not RSA public key")
	}

	return rsaKey, nil
}

// loadPrivateKeyFromFile загружает приватный ключ из файла
func loadPrivateKeyFromFile(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	return parsePrivateKey(data)
}

// loadPublicKeyFromFile загружает публичный ключ из файла
func loadPublicKeyFromFile(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}
	return parsePublicKey(data)
}

