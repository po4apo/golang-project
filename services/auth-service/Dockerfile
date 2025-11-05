# Этап сборки
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum из корня проекта
COPY go.mod go.sum ./

# Копируем весь проект (нужен для локальных импортов)
COPY . .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарник auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth-service ./services/auth-service/cmd/main.go

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates postgresql-client wget

WORKDIR /root/

# Копируем бинарник из этапа сборки
COPY --from=builder /app/auth-service .

# Копируем миграции
COPY --from=builder /app/services/auth-service/migrations ./migrations

# Копируем entrypoint скрипт
COPY --from=builder /app/services/auth-service/entrypoint.sh .
RUN chmod +x entrypoint.sh

# Открываем порт gRPC
EXPOSE 50051

# Запускаем через entrypoint
ENTRYPOINT ["./entrypoint.sh"]

