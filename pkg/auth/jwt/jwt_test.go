package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"testing"
	"time"
)

// generateTestKeys создаёт тестовую пару RSA ключей
func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

// privateKeyToPEM конвертирует приватный ключ в PEM формат
func privateKeyToPEM(key *rsa.PrivateKey) []byte {
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}
	return pem.EncodeToMemory(block)
}

// publicKeyToPEM конвертирует публичный ключ в PEM формат
func publicKeyToPEM(key *rsa.PublicKey) []byte {
	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		panic(err)
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keyBytes,
	}
	return pem.EncodeToMemory(block)
}

func TestNewManager(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with PEM strings",
			config: Config{
				PrivateKey: string(privateKeyPEM),
				PublicKey:  string(publicKeyPEM),
				Issuer:     "test-issuer",
				TTL:        time.Hour,
			},
			wantErr: false,
		},
		{
			name: "valid config without public key (extracted from private)",
			config: Config{
				PrivateKey: string(privateKeyPEM),
				Issuer:     "test-issuer",
				TTL:        time.Hour,
			},
			wantErr: false,
		},
		{
			name: "missing private key",
			config: Config{
				PublicKey: string(publicKeyPEM),
				Issuer:    "test-issuer",
				TTL:       time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid private key",
			config: Config{
				PrivateKey: "invalid key",
				PublicKey:  string(publicKeyPEM),
				Issuer:     "test-issuer",
				TTL:        time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid public key",
			config: Config{
				PrivateKey: string(privateKeyPEM),
				PublicKey:  "invalid key",
				Issuer:     "test-issuer",
				TTL:        time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && manager == nil {
				t.Error("NewManager() returned nil manager without error")
			}
		})
	}
}

func TestManager_Sign(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	manager, err := NewManager(Config{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
		Issuer:     "test-issuer",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	token, err := manager.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if token == "" {
		t.Error("Sign() returned empty token")
	}

	// Проверка, что токен можно валидировать
	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("Validate() UserID = %v, want user-123", claims.UserID)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Validate() Issuer = %v, want test-issuer", claims.Issuer)
	}
}

func TestManager_Validate_InvalidToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	manager, err := NewManager(Config{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
		Issuer:     "test-issuer",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "invalid format",
			token:   "not.a.valid.token",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "invalid",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "token with wrong algorithm (HS256)",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			wantErr: ErrInvalidSigningMethod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.Validate(tt.token)
			if err == nil {
				t.Error("Validate() expected error, got nil")
				return
			}
			if !errors.Is(err, tt.wantErr) && err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_Validate_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	// Создаём manager с очень коротким TTL
	manager, err := NewManager(Config{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
		Issuer:     "test-issuer",
		TTL:        1 * time.Nanosecond, // Очень короткий TTL
	})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Генерируем токен
	token, err := manager.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Ждём истечения токена
	time.Sleep(10 * time.Millisecond)

	// Пытаемся валидировать истёкший токен
	_, err = manager.Validate(token)
	if err == nil {
		t.Error("Validate() expected error for expired token, got nil")
		return
	}
	if err != ErrExpiredToken {
		t.Errorf("Validate() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestManager_Validate_WrongIssuer(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	// Создаём manager с issuer "issuer-1"
	manager1, err := NewManager(Config{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
		Issuer:     "issuer-1",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create manager1: %v", err)
	}

	// Создаём manager с issuer "issuer-2"
	manager2, err := NewManager(Config{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
		Issuer:     "issuer-2",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create manager2: %v", err)
	}

	// Генерируем токен с manager1
	token, err := manager1.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Пытаемся валидировать токен с manager2 (другой issuer)
	_, err = manager2.Validate(token)
	if err == nil {
		t.Error("Validate() expected error for wrong issuer, got nil")
		return
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestManager_Validate_WrongKey(t *testing.T) {
	// Генерируем две разные пары ключей
	privateKey1, publicKey1 := generateTestKeys(t)
	_, publicKey2 := generateTestKeys(t)

	privateKey1PEM := privateKeyToPEM(privateKey1)
	publicKey1PEM := publicKeyToPEM(publicKey1)
	publicKey2PEM := publicKeyToPEM(publicKey2)

	// Создаём manager1 для подписи
	manager1, err := NewManager(Config{
		PrivateKey: string(privateKey1PEM),
		PublicKey:  string(publicKey1PEM),
		Issuer:     "test-issuer",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create manager1: %v", err)
	}

	// Создаём manager2 с другим публичным ключом для валидации
	manager2, err := NewManager(Config{
		PrivateKey: string(privateKey1PEM), // Используем тот же приватный для создания manager
		PublicKey:  string(publicKey2PEM), // Но другой публичный для валидации
		Issuer:     "test-issuer",
		TTL:        time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create manager2: %v", err)
	}

	// Генерируем токен с manager1
	token, err := manager1.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Пытаемся валидировать токен с manager2 (другой публичный ключ)
	_, err = manager2.Validate(token)
	if err == nil {
		t.Error("Validate() expected error for wrong public key, got nil")
		return
	}
	// Ожидаем ошибку валидации подписи
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Validate() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestNewManagerFromEnv(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	privateKeyPEM := privateKeyToPEM(privateKey)
	publicKeyPEM := publicKeyToPEM(publicKey)

	// Сохраняем оригинальные значения
	origPrivateKey := os.Getenv("JWT_RSA_PRIVATE_KEY")
	origPublicKey := os.Getenv("JWT_RSA_PUBLIC_KEY")
	origIssuer := os.Getenv("JWT_ISSUER")
	origTTL := os.Getenv("JWT_TTL")

	// Устанавливаем тестовые значения
	os.Setenv("JWT_RSA_PRIVATE_KEY", string(privateKeyPEM))
	os.Setenv("JWT_RSA_PUBLIC_KEY", string(publicKeyPEM))
	os.Setenv("JWT_ISSUER", "test-issuer")
	os.Setenv("JWT_TTL", "1h")

	// Восстанавливаем после теста
	defer func() {
		if origPrivateKey != "" {
			os.Setenv("JWT_RSA_PRIVATE_KEY", origPrivateKey)
		} else {
			os.Unsetenv("JWT_RSA_PRIVATE_KEY")
		}
		if origPublicKey != "" {
			os.Setenv("JWT_RSA_PUBLIC_KEY", origPublicKey)
		} else {
			os.Unsetenv("JWT_RSA_PUBLIC_KEY")
		}
		if origIssuer != "" {
			os.Setenv("JWT_ISSUER", origIssuer)
		} else {
			os.Unsetenv("JWT_ISSUER")
		}
		if origTTL != "" {
			os.Setenv("JWT_TTL", origTTL)
		} else {
			os.Unsetenv("JWT_TTL")
		}
	}()

	manager, err := NewManagerFromEnv()
	if err != nil {
		t.Fatalf("NewManagerFromEnv() error = %v", err)
	}
	if manager == nil {
		t.Error("NewManagerFromEnv() returned nil manager")
	}

	// Проверяем, что manager работает
	token, err := manager.Sign("user-123")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("Validate() UserID = %v, want user-123", claims.UserID)
	}
}

func TestNewManagerFromEnv_MissingKey(t *testing.T) {
	// Сохраняем оригинальные значения
	origPrivateKey := os.Getenv("JWT_RSA_PRIVATE_KEY")
	origPublicKey := os.Getenv("JWT_RSA_PUBLIC_KEY")

	// Удаляем переменные окружения
	os.Unsetenv("JWT_RSA_PRIVATE_KEY")
	os.Unsetenv("JWT_RSA_PUBLIC_KEY")

	// Восстанавливаем после теста
	defer func() {
		if origPrivateKey != "" {
			os.Setenv("JWT_RSA_PRIVATE_KEY", origPrivateKey)
		}
		if origPublicKey != "" {
			os.Setenv("JWT_RSA_PUBLIC_KEY", origPublicKey)
		}
	}()

	_, err := NewManagerFromEnv()
	if err == nil {
		t.Error("NewManagerFromEnv() expected error for missing key, got nil")
	}
}

