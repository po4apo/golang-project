# Docker Compose для Golang Microservices Project

## Быстрый старт

### Запуск всех сервисов:

```bash
# Из корня проекта
make docker-up

# Или напрямую
cd deploy/docker-compose
docker-compose up -d
```

### Проверка статуса:

```bash
docker-compose ps
```

### Просмотр логов:

```bash
# Все сервисы
make docker-logs

# Конкретный сервис
docker-compose logs -f auth-service
docker-compose logs -f rest-api
```

### Остановка сервисов:

```bash
make docker-down
```

### Перезапуск:

```bash
make docker-restart
```

### Полная очистка (удаление контейнеров, томов и образов):

```bash
make docker-clean
```

## Архитектура

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP :8080
       ▼
┌─────────────┐
│  REST API   │
└──────┬──────┘
       │ gRPC :50051
       ▼
┌─────────────┐      ┌──────────────┐
│auth-service │─────▶│ PostgreSQL   │
└─────────────┘      │   :5432      │
                     └──────────────┘
```

## Сервисы

### postgres-auth
- **Образ**: `postgres:15-alpine`
- **Порт**: `5432`
- **База**: `authdb`
- **Пользователь**: `authuser`
- **Пароль**: `authpass`

### auth-service
- **Порт**: `50051` (gRPC)
- **Миграции**: применяются автоматически при старте
- **Зависимости**: postgres-auth

### rest-api
- **Порт**: `8080` (HTTP)
- **Endpoints**:
  - `GET /health` - healthcheck
  - `POST /api/v1/auth/signup` - регистрация
  - `POST /api/v1/auth/signin` - вход
  - `GET /api/v1/auth/validate` - проверка токена
- **Зависимости**: auth-service

## Тестирование

После запуска сервисов:

```bash
# Healthcheck
curl http://localhost:8080/health

# Регистрация
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}'

# Вход
curl -X POST http://localhost:8080/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}'

# Проверка токена
curl -X GET http://localhost:8080/api/v1/auth/validate \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Переменные окружения

Переменные можно изменить в `docker-compose.yaml`:

### auth-service
- `GRPC_ADDR` - адрес gRPC сервера (по умолчанию: `:50051`)
- `DB_DSN` - строка подключения к PostgreSQL
- `JWT_KEY` - секретный ключ для JWT
- `LOG_LEVEL` - уровень логирования (`debug`, `info`, `warn`, `error`)

### rest-api
- `HTTP_ADDR` - адрес HTTP сервера (по умолчанию: `:8080`)
- `AUTH_GRPC_ADDR` - адрес auth-service (по умолчанию: `auth-service:50051`)
- `LOG_LEVEL` - уровень логирования

## Разработка

### Пересборка образов:

```bash
make docker-build
make docker-up
```

### Применение миграций вручную:

```bash
# Войти в контейнер
docker exec -it auth-service sh

# Проверить статус миграций
migrate -database "$DB_DSN" -path ./migrations version
```

## Troubleshooting

### Проблемы с портами

Если порты заняты, остановите локальные сервисы:
```bash
# Найти процессы на портах
lsof -i :8080
lsof -i :50051
lsof -i :5432

# Убить процесс
kill -9 <PID>
```

### Проблемы с миграциями

```bash
# Посмотреть логи auth-service
docker-compose logs auth-service

# Войти в контейнер и проверить миграции
docker exec -it auth-service sh
ls -la migrations/
```

### Очистка и пересоздание

```bash
# Полная очистка
make docker-clean

# Пересоздание
make docker-build
make docker-up
```




