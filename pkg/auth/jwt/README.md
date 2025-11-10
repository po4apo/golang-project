# JWT Manager (RS256)

Пакет для работы с JWT токенами с использованием алгоритма RS256 (RSA).

## Возможности

- ✅ Генерация JWT токенов с RS256
- ✅ Валидация JWT токенов
- ✅ Поддержка загрузки ключей из переменных окружения
- ✅ Поддержка загрузки ключей из файлов
- ✅ Автоматическое извлечение публичного ключа из приватного
- ✅ Проверка срока действия токенов
- ✅ Проверка issuer
- ✅ Покрытие негативных кейсов тестами

## Использование

### 1. Генерация RSA ключей

```bash
# Используйте скрипт для генерации ключей
./scripts/generate-jwt-keys.sh

# Или вручную:
openssl genrsa -out jwt_private_key.pem 2048
openssl rsa -in jwt_private_key.pem -pubout -out jwt_public_key.pem
```

### 2. Настройка переменных окружения

#### Вариант 1: Содержимое ключей (для Kubernetes secrets)

```bash
export JWT_RSA_PRIVATE_KEY="$(cat jwt_private_key.pem)"
export JWT_RSA_PUBLIC_KEY="$(cat jwt_public_key.pem)"
export JWT_ISSUER="auth-service"
export JWT_TTL="24h"
```

#### Вариант 2: Путь к файлам

```bash
export JWT_RSA_PRIVATE_KEY_PATH="./jwt_private_key.pem"
export JWT_RSA_PUBLIC_KEY_PATH="./jwt_public_key.pem"
export JWT_ISSUER="auth-service"
export JWT_TTL="24h"
```

### 3. Использование в коде

```go
import "golang-project/pkg/auth/jwt"

// Создание manager из переменных окружения
manager, err := jwt.NewManagerFromEnv()
if err != nil {
    log.Fatalf("failed to initialize JWT manager: %v", err)
}

// Или с явной конфигурацией
manager, err := jwt.NewManager(jwt.Config{
    PrivateKey: string(privateKeyPEM),
    PublicKey:  string(publicKeyPEM),
    Issuer:     "auth-service",
    TTL:        24 * time.Hour,
})

// Генерация токена
token, err := manager.Sign("user-123")
if err != nil {
    return err
}

// Валидация токена
claims, err := manager.Validate(token)
if err != nil {
    if err == jwt.ErrExpiredToken {
        // Токен истёк
    } else if err == jwt.ErrInvalidToken {
        // Токен невалиден
    }
    return err
}

// Использование claims
userID := claims.UserID
```

## Структура Claims

```go
type Claims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}
```

Claims содержит:
- `UserID` - ID пользователя
- `Issuer` - издатель токена
- `IssuedAt` - время создания
- `ExpiresAt` - время истечения

## Обработка ошибок

Пакет возвращает следующие ошибки:

- `ErrInvalidToken` - токен невалиден (неправильный формат, подпись, issuer)
- `ErrExpiredToken` - токен истёк
- `ErrInvalidSigningMethod` - используется неправильный алгоритм подписи
- `ErrMissingKey` - отсутствует необходимый ключ

## Тестирование

```bash
go test ./pkg/auth/jwt -v
```

Все негативные кейсы покрыты тестами:
- ✅ Невалидные токены
- ✅ Истёкшие токены
- ✅ Неправильный issuer
- ✅ Неправильный публичный ключ
- ✅ Неправильный алгоритм подписи
- ✅ Отсутствующие ключи

## Безопасность

⚠️ **Важно:**
- Приватный ключ должен храниться в секретах (Kubernetes secrets, HashiCorp Vault и т.д.)
- Публичный ключ можно распространять для валидации токенов
- Используйте ключи длиной минимум 2048 бит
- Регулярно ротируйте ключи

## Интеграция с Kubernetes

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: jwt-keys
type: Opaque
stringData:
  private-key: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  public-key: |
    -----BEGIN PUBLIC KEY-----
    ...
    -----END PUBLIC KEY-----
```

```yaml
env:
  - name: JWT_RSA_PRIVATE_KEY
    valueFrom:
      secretKeyRef:
        name: jwt-keys
        key: private-key
  - name: JWT_RSA_PUBLIC_KEY
    valueFrom:
      secretKeyRef:
        name: jwt-keys
        key: public-key
  - name: JWT_ISSUER
    value: "auth-service"
  - name: JWT_TTL
    value: "24h"
```

