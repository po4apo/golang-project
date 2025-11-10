# Этап сборки
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum из корня проекта
COPY go.mod go.sum ./

# Копируем весь проект (нужен для локальных импортов)
COPY . .

# Устанавливаем зависимости
RUN go mod download

# Собираем бинарник rest-api
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rest-api ./services/rest-api/cmd/main.go

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из этапа сборки
COPY --from=builder /app/rest-api .

# Открываем порт HTTP
EXPOSE 8080

# Запускаем сервис
CMD ["./rest-api"]







