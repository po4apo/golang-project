<!-- 188b3434-5575-4990-9c8c-8538e07bbecc 112e8f3b-dbb1-4c9d-b885-604b8d7efd5f -->
# Архитектура и план: ToDo/Notes на Go — gRPC микросервисы + REST BFF + K8s + Saga/Outbox

## Технологии и стандарты

- **Язык**: Go 1.22
- **Коммуникации**: gRPC (+ protoc-gen-validate), отдельный REST BFF (chi)
- **Контракты**: Protocol Buffers с `buf` (lint, breaking checks)
- **БД**: PostgreSQL (каждый сервис — своя схема/инстанс), миграции `golang-migrate`
- **Сообщения**: Kafka (bitnami локально; в K8s — Bitnami/Strimzi), клиент `segmentio/kafka-go`
- **Шаблоны**: Outbox (транзакционная таблица + ретранслятор); Saga (оркестратор)
- **Конфигурация**: 12factor, env + Viper, `config.yaml` как base
- **Логирование**: zerolog (JSON), **трейсинг**: OpenTelemetry + OTLP (Jaeger), **метрики**: Prometheus
- **Безопасность**: JWT (RS256), Argon2 пароли, gRPC TLS между сервисами
- **Качество**: golangci-lint, `buf lint`/`buf breaking`, unit + integration, mockery
- **Сборка**: Docker multi-stage, Makefile, CI (GitHub Actions)
- **Деплой**: Kubernetes (Helm/Kustomize), HPA, Ingress, ServiceMonitor

## Сервисы и ответственность

- **auth-service**: регистрация/логин, JWT, события `UserCreated`
- **notes-service**: CRUD заметок, события `NoteCreated`
- **bff-rest**: внешний HTTP API; аутентификация (JWT), валидация, маппинг на gRPC, Swagger/OpenAPI
- **saga-orchestrator**: оркестрация бизнес-процессов (SignUpSaga, DeleteUserSaga): вызовы gRPC + ожидание ответов, публикация команд/ивентов

## События, Outbox и Saga

- **Outbox**: в каждом сервисе таблица `outbox` + транзакционная запись события вместе с бизнес-сущностью; фоновый ретранслятор публикует в Kafka и помечает `sent_at`.
- **Темы Kafka**: `auth.events`, `notes.events`, `saga.commands`, `saga.replies`, `dlq`.
- **Идемпотентность**: ключ сообщения = `aggregate_id`/`event_id`; консюмер хранит `processed_event_id`.
- **Сага (оркестрация)**: отдельный сервис хранит состояние саги (таблица `sagas`), корреляция по `correlation_id`, ретраи/таймауты/компенсации.
- Пример: `SignUpSaga`: Step1 `AuthService.SignUp` → Step2 `NotesService.Create` (welcome-note); при фейле Step2 — компенсация `AuthService.DeleteUser`.

### Схемы событий (protobuf)

Формат «envelope + payload»:

```proto
syntax = "proto3";
package common.v1;
option go_package = "github.com/your-org/notes/proto/gen/go/common/v1;commonv1";

message EventEnvelope {
  string event_id = 1;        // UUID
  string aggregate_id = 2;    // user_id / note_id
  string type = 3;            // e.g. "auth.UserCreated"
  int64  occurred_at = 4;     // unix millis
  bytes  payload = 5;         // marshaled event (protobuf)
  string correlation_id = 6;  // saga id
}

message UserCreated { string user_id = 1; string email = 2; }
message NoteCreated { string note_id = 1; string owner_id = 2; }
```

### Outbox таблица (пример)

```sql
CREATE TABLE outbox (
  id UUID PRIMARY KEY,
  aggregate_id TEXT NOT NULL,
  type TEXT NOT NULL,
  payload BYTEA NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sent_at TIMESTAMPTZ NULL,
  correlation_id TEXT NULL
);
CREATE INDEX idx_outbox_sent_at ON outbox (sent_at);
```

## Быстрый "walking skeleton"

- Сквозные маршруты: `POST /v1/auth/sign-up`, `POST /v1/auth/sign-in`, `POST /v1/notes`, `GET /v1/notes`
- Плюс минимальная сага `SignUpSaga` (после успешного `sign-up` создаёт welcome-note).
- Локально: `docker compose up` — Postgres x2, Kafka + Zookeeper + kafka-ui, Jaeger, auth, notes, bff, saga-orchestrator.
- Наблюдаемость: `/healthz`, трассы, метрики, логи с trace_id.

## Структура репозитория

- `/proto/`
  - `/proto/auth/v1/auth.proto`
  - `/proto/notes/v1/notes.proto`
  - `/proto/common/v1/events.proto`
  - `/proto/buf.yaml`, `/proto/buf.gen.yaml`
- `/services/`
  - `/services/auth-service/` — `cmd/server/main.go`, `internal/...`, `migrations/`, `Dockerfile`
  - `/services/notes-service/` — аналогично
  - `/services/bff-rest/` — `cmd/server/main.go`, `internal/http/handlers`, `internal/clients/grpc`, `openapi/`, `Dockerfile`
  - `/services/saga-orchestrator/` — `cmd/server/main.go`, `internal/saga`, `internal/bus`, `Dockerfile`
- `/pkg/` — `logger`, `config`, `otel`, `auth/jwt`, `outbox`, `kafka`, `saga`
- `/deploy/`
  - `/deploy/docker-compose/`
  - `/deploy/k8s/` (Helm chart или Kustomize)
- `Makefile`, `golangci.yaml`, `buf.work.yaml`, `.github/workflows/ci.yaml`

## REST BFF (ключевые эндпоинты)

- `POST /v1/auth/sign-up` → `AuthService.SignUp` (+ публикация `UserCreated` из outbox в auth)
- `POST /v1/auth/sign-in` → `AuthService.SignIn`
- `GET /v1/notes` → `NotesService.List` (по `user_id` из JWT)
- `POST /v1/notes` → `NotesService.Create` (публикация `NoteCreated` в notes)
- `GET /v1/notes/{id}` → `NotesService.Get`
- `PUT /v1/notes/{id}` → `NotesService.Update`
- `DELETE /v1/notes/{id}` → `NotesService.Delete`

Минимальная маршрутизация BFF (псевдо):

```go
r.Group(func(r chi.Router) {
  r.Post("/v1/auth/sign-up", h.SignUp)
  r.Post("/v1/auth/sign-in", h.SignIn)
})

r.Group(func(r chi.Router) {
  r.Use(JWTMiddleware)
  r.Get("/v1/notes", h.ListNotes)
  r.Post("/v1/notes", h.CreateNote)
  r.Get("/v1/notes/{id}", h.GetNote)
  r.Put("/v1/notes/{id}", h.UpdateNote)
  r.Delete("/v1/notes/{id}", h.DeleteNote)
})
```

## Данные и миграции

- auth-service: `users(id uuid, email unique, pass_hash, created_at)`, `outbox(...)`
- notes-service: `notes(id uuid, owner_id uuid, title, content, created_at, updated_at, deleted_at null)`, `outbox(...)`
- saga-orchestrator: `sagas(id uuid, state, step, correlation_id, updated_at)`
- Миграции в `/services/*/migrations` + `make migrate-<svc>`

## Наблюдаемость и эксплуатация

- Health: gRPC health-check + `/healthz` в BFF и orchestrator
- Tracing: экспорт в Jaeger/Tempo, parent trace → BFF → gRPC → Kafka (trace context в headers)
- Metrics: `/metrics`, gRPC и Kafka middlewares для Prometheus
- Логи: корреляция по `trace_id`/`span_id`/`correlation_id`

## Локальная разработка

- `deploy/docker-compose/docker-compose.yaml`: postgres-auth, postgres-notes, kafka, zookeeper, kafka-ui, jaeger, auth, notes, bff, saga-orchestrator
- Makefile: `make generate` (buf), `make lint`, `make test`, `make dev`, `make build`, `make migrate-*`, `make up`

## Kubernetes (минимум для старта)

- Helm chart `deploy/helm/notes-platform/`:
  - `values.yaml` для образов/секретов/ресурсов (вкл. Kafka брокер url, топики)
  - Шаблоны: `Deployment`, `Service`, `ConfigMap`, `Secret`, `HPA`, `Ingress`, `ServiceMonitor`
- Kafka: dev — Bitnami Chart внутри кластера; prod — управляемый кластер или Strimzi
- Секреты: `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`, `DB_DSN`, `KAFKA_BROKERS`
- Ingress: публичный только для `bff-rest`; внутренние — ClusterIP

## CI/CD

- GitHub Actions: job "lint-test" (go + buf + golangci), job "docker" (buildx + push), job "compose-test" (поднимает Kafka+DB и гоняет интеграционные тесты), job "helm" (optional)

## План релизов (итерации)

1) Walking skeleton: аутентификация + список/создание заметок, локально + Docker

2) Внедрить Outbox в auth/notes и ретранслятор; добавить Kafka

3) Реализовать `SignUpSaga` в оркестраторе (welcome-note); e2e тесты

4) Полный CRUD заметок, refresh-токены, пагинация, Swagger BFF

5) Обсервабилити, rate limit, ретраи, таймауты, TLS gRPC

6) Kubernetes: dev namespace, затем prod

### To-dos

- [ ] Создать скелет репо, Makefile, общие пакеты, golangci/buf
- [ ] Описать контракты auth/notes в proto и buf
- [ ] Настроить buf.gen, protoc-plugins, generate таргеты
- [ ] Реализовать auth-service с БД, миграциями, JWT
- [ ] Реализовать notes-service с БД, миграциями
- [ ] Собрать REST BFF c chi, JWT миддлварь, gRPC клиенты
- [ ] Dockerfiles и docker-compose для локалки
- [ ] Включить OTel, Prometheus, health checks
- [ ] Подготовить Helm/Kustomize манифесты
- [ ] GitHub Actions: lint/test/build/docker/buf


