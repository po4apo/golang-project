# Декомпозиция: как шаг за шагом собрать сервис

Учебный чек-лист по материалам `docs/architecture.md` и `docs/plan.md`. Для каждого шага указаны действия (минимум) и критерии готовности (DoD).

## Этап 0. Подготовка окружения

- [x] Установить инструменты
  - Действия: Go 1.22+, Docker/Compose, buf, protoc, golangci-lint, make, mockery.
  - DoD: `go version`, `docker compose version`, `buf --version`, `protoc --version` выводятся корректно.
- [x] Инициализация репозитория
  - Действия: `git init`, `.gitignore`, базовый `README.md`.
  - DoD: первый коммит с пустым скелетом.

## Этап A. Контракты и генерация кода

- [x] Buf workspace и линтеры
  - Действия: `proto/buf.yaml`, `buf.work.yaml` (при необходимости), `buf lint`/`buf breaking`.
  - DoD: `buf lint` проходит.
- [x] Описать proto контракты
  - Действия: `proto/auth/v1/auth.proto`, `proto/notes/v1/notes.proto`, `proto/common/v1/events.proto`.
  - DoD: `buf build` успешен.
- [x] Настроить codegen
  - Действия: `proto/buf.gen.yaml`, плагины gRPC/validators, `make generate`.
  - DoD: сгенерированы `proto/gen/go/...`, проект собирается.

## Этап B. auth-service

- [x] Скелет сервиса
  - Действия: `cmd/server/main.go`, пакеты `internal/...`.
  - DoD: gRPC-сервер стартует.
- [x] Конфигурация/логирование
  - Действия: `pkg/config` (Viper), `pkg/logger` (slog).
  - DoD: JSON-логи; конфиг из env.
- [x] Миграции БД
  - Действия: таблицы `users`, `outbox`; `golang-migrate`.
  - DoD: `make migrate-auth up` создаёт таблицы.
- [x] Домен и безопасность
  - Действия: репозиторий пользователей, Argon2, валидация email/пароля.
  - DoD: unit-тесты на создание и хеширование.
- [x] gRPC методы
  - Действия: SignUp/SignIn/ValidateToken, корректные статус-коды gRPC.
  - DoD: хэндлеры работают (SignUp/SignIn протестированы через grpcurl).
- [ ] JWT (RS256)
  - Действия: `pkg/auth/jwt`, ключи из секретов.
  - DoD: токен валидируется, негативные кейсы покрыты.

## Этап C. notes-service

- [ ] Скелет/конфигурация
  - Действия: поднять gRPC сервер.
  - DoD: сервер стартует.
- [ ] Миграции/репозитории
  - Действия: таблицы `notes`, `outbox`; CRUD; пагинация.
  - DoD: миграции применяются; CRUD-тесты зелёные.
- [ ] gRPC CRUD
  - Действия: Create/Get/List/Update/Delete; проверки owner_id.
  - DoD: интеграционные тесты API.

## Этап D. Outbox и Kafka

- [ ] Пакет outbox
  - Действия: транзакционная запись события, envelope.
  - DoD: запись в outbox вместе с бизнес-транзакцией.
- [ ] Ретранслятор outbox → Kafka
  - Действия: воркер, батчи, публикация, `sent_at`, backoff.
  - DoD: события в `auth.events`/`notes.events` публикуются.
- [ ] Kafka локально
  - Действия: compose: брокер, zookeeper, kafka-ui; создать темы.
  - DoD: продюс/консюм работает.
- [ ] Идемпотентность
  - Действия: хранить `processed_event_id`, ключи сообщений.
  - DoD: дубликаты не нарушают инварианты (интеграционный тест).

## Этап E. BFF REST

- [ ] Скелет сервера
  - Действия: chi, `/healthz`, `/metrics`.
  - DoD: сервер отвечает 200 на healthz.
- [ ] JWT-мидлварь
  - Действия: `Authorization: Bearer`, извлечь `user_id`.
  - DoD: защищённые ручки требуют валидный токен.
- [ ] Прокси к gRPC
  - Действия: клиенты auth/notes, маршруты из плана.
  - DoD: корректная трансляция ошибок.
- [ ] Swagger/OpenAPI
  - Действия: описать основные ручки.
  - DoD: Swagger доступен локально.

## Этап F. Сага (saga-orchestrator)

- [ ] Скелет/состояние
  - Действия: сервис оркестратора, таблица `sagas`.
  - DoD: сервис стартует; DAO состояния есть.
- [ ] SignUpSaga — happy path
  - Действия: на `UserCreated` вызвать `NotesService.Create` (welcome-note).
  - DoD: e2e — после sign-up появляется заметка.
- [ ] Отказы/компенсации
  - Действия: таймауты, ретраи, `AuthService.DeleteUser` при фейле.
  - DoD: e2e с инъекцией ошибки завершает компенсацией.
- [ ] Команды/ответы (опц.)
  - Действия: `saga.commands`/`saga.replies` вместо прямого gRPC.
  - DoD: альтернативный путь работает локально.

## Этап G. Наблюдаемость

- [ ] Трейсинг OTel
  - Действия: провайдер, Jaeger, пропагация через gRPC/Kafka headers.
  - DoD: в Jaeger видна цепочка BFF → gRPC → Kafka → Orchestrator.
- [ ] Метрики Prometheus
  - Действия: http `/metrics`, gRPC/Kafka middlewares.
  - DoD: метрики считываются.
- [ ] Логи
  - Действия: JSON, корреляция `trace_id`/`correlation_id`.
  - DoD: связанный поток виден в логах.

## Этап H. Docker/Compose

- [ ] Dockerfiles (multi-stage)
  - Действия: для всех сервисов.
  - DoD: образы собираются и стартуют.
- [ ] docker-compose
  - Действия: Postgres x2, Kafka, kafka-ui, Jaeger, сервисы.
  - DoD: `docker compose up` поднимает стек; healthz зелёные.
- [ ] Makefile
  - Действия: `generate`, `lint`, `test`, `build`, `up/down`, `migrate-*`.
  - DoD: команды работают на чистой машине.

## Этап I. Kubernetes (dev)

- [ ] Манифесты/Helm/Kustomize
  - Действия: Deployment/Service/Ingress/HPA/ConfigMap/Secret/ServiceMonitor.
  - DoD: dev namespace развёрнут.
- [ ] Секреты/конфиги
  - Действия: `JWT_*`, `DB_DSN`, `KAFKA_BROKERS`.
  - DoD: Pod-ы читают секреты; readiness/liveness ок.
- [ ] Обсервабилити в кластере
  - Действия: scrape Prometheus, Jaeger, Ingress BFF.
  - DoD: BFF доступен; метрики/трейсы видны.

## Этап J. CI/CD

- [ ] Lint/Test
  - Действия: GH Actions: кеш Go, `golangci-lint`, unit, `buf lint/breaking`.
  - DoD: PR падает при ошибках.
- [ ] Docker build/push
  - Действия: buildx, matrix по сервисам, теги SHA/semver.
  - DoD: образы в реестре.
- [ ] Compose-test job
  - Действия: поднимает зависимости, гоняет интеграционные/e2e саги.
  - DoD: job зелёный — обязательный для merge.
- [ ] (Опц.) Helm release
  - Действия: helm lint/release.
  - DoD: артефакты релиза доступны.

---

## Навигация по документации

- Схемы БД, события, последовательности: `docs/architecture.md`.
- Обзор, структура, маршруты, релиз-план: `docs/plan.md`.

## Рекомендации

- Выполнять этапы последовательно, не распараллеливать A–E.
- Каждый шаг завершать DoD и минимальными тестами.
- Фиксировать решения и договорённости в README сервисов.

## Конкретизация по шагам (команды, файлы, DoD)

### Этап 0 — Подготовка

- [ ] Установка инструментов (Ubuntu/WSL)
  - Команды:
    ```bash
    sudo apt update && sudo apt install -y git make protobuf-compiler
    go install github.com/bufbuild/buf/cmd/buf@latest
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    go install github.com/envoyproxy/protoc-gen-validate@latest
    go install github.com/vektra/mockery/v2@latest
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
    ```
  - DoD: все бинарники в `$(go env GOPATH)/bin`, `protoc --version` и `buf --version` работают.

- [ ] Инициализация репозитория
  - Команды:
    ```bash
    git init && git add . && git commit -m "chore: init repo"
    ```
  - DoD: первый коммит создан.

### Этап A — Контракты и генерация

- [ ] Структура proto и buf
  - Команды:
    ```bash
    mkdir -p proto/{auth/v1,notes/v1,common/v1}
    ```
  - Файлы (минимум):
    - `proto/buf.yaml`:
      ```yaml
      version: v1
      lint:
        use:
          - DEFAULT
      ```
    - `proto/buf.gen.yaml`:
      ```yaml
      version: v1
      plugins:
        - name: go
          out: gen/go
          opt: paths=source_relative
        - name: go-grpc
          out: gen/go
          opt: paths=source_relative
        - name: validate
          out: gen/go
          opt: lang=go,paths=source_relative
      ```
  - DoD: `buf lint`/`buf generate` проходят; появляется `proto/gen/go/...`.

- [ ] Makefile цели
  - Файл `Makefile` (фрагмент):
    ```make
    .PHONY: generate lint test
    generate:
    	cd proto && buf generate
    lint:
    	golangci-lint run ./...
    test:
    	go test ./...
    ```
  - DoD: `make generate` создает код, `make lint`/`make test` выполняются.

### Этап B — auth-service

- [ ] Зависимости
  - Команды:
    ```bash
    go get google.golang.org/grpc
    go get github.com/spf13/viper
    go get github.com/rs/zerolog
    go get github.com/jackc/pgx/v5
    go get github.com/golang-jwt/jwt/v5
    ```
  - DoD: `go mod tidy` без ошибок.

- [ ] Скелет сервера
  - Файлы:
    - `services/auth-service/cmd/server/main.go`
    - `services/auth-service/internal/transport/grpc/server.go`
    - `services/auth-service/internal/domain/user.go`
    - `services/auth-service/internal/repo/user_repo.go`
    - `services/auth-service/internal/app/service.go`
  - DoD: `go run services/auth-service/cmd/server/main.go` поднимает gRPC порт (лог в JSON).

- [ ] Миграции БД
  - Файлы:
    - `services/auth-service/migrations/0001_init_users.sql`
    - `services/auth-service/migrations/0002_init_outbox.sql`
  - Команда применения (пример):
    ```bash
    migrate -database "$AUTH_DB_DSN" -path services/auth-service/migrations up
    ```
  - DoD: таблицы `users`, `outbox` созданы.

- [ ] Реализация SignUp/SignIn/ValidateToken
  - Действия: маппинг protobuf ↔ домен, статусы ошибок через `status.Error`.
  - DoD: юнит-тесты хэндлеров; `SignUp` создаёт пользователя, `SignIn` выдаёт JWT.

- [ ] JWT (RS256)
  - Файлы:
    - `pkg/auth/jwt/jwt.go` (генерация/валидация)
  - Переменные окружения: `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`.
  - DoD: токен подписывается и валидируется; истёкший токен отклоняется.

### Этап C — notes-service

- [ ] Скелет, миграции и CRUD
  - Файлы:
    - `services/notes-service/cmd/server/main.go`
    - `services/notes-service/internal/...` (аналогично auth)
    - `services/notes-service/migrations/0001_init_notes.sql`
    - `services/notes-service/migrations/0002_init_outbox.sql`
  - DoD: gRPC сервер поднят; CRUD-тесты зелёные.

### Этап D — Outbox и Kafka

- [ ] Пакеты инфраструктуры
  - Файлы:
    - `pkg/outbox/outbox.go` (интерфейс и запись события в TX)
    - `pkg/kafka/producer.go` (инициализация и публикация)
    - `pkg/kafka/consumer.go` (подписка, обработка, коммиты)
  - DoD: запись в outbox в рамках транзакции; продюсер публикует в нужные топики.

- [ ] Ретранслятор outbox
  - Файлы:
    - `services/auth-service/internal/outbox/relay.go`
    - `services/notes-service/internal/outbox/relay.go`
  - Логика: периодическая выборка `sent_at IS NULL`, паблиш, update `sent_at`; backoff.
  - DoD: события попадают в `auth.events`/`notes.events`.

- [ ] Docker Compose для Kafka
  - Файл: `deploy/docker-compose/docker-compose.yaml` (фрагмент):
    ```yaml
    services:
      zookeeper:
        image: bitnami/zookeeper:latest
        environment:
          - ALLOW_ANONYMOUS_LOGIN=yes
      kafka:
        image: bitnami/kafka:latest
        environment:
          - KAFKA_CFG_ZOOKEEPER_CONNECT=zookeeper:2181
          - ALLOW_PLAINTEXT_LISTENER=yes
        ports:
          - "9092:9092"
      kafka-ui:
        image: provectuslabs/kafka-ui:latest
        ports:
          - "8082:8080"
    ```
  - DoD: UI доступен, топики видны.

### Этап E — BFF REST

- [ ] Зависимости и сервер
  - Команды:
    ```bash
    go get github.com/go-chi/chi/v5
    go get github.com/go-chi/jwtauth/v5
    ```
  - Файлы:
    - `services/bff-rest/cmd/server/main.go`
    - `services/bff-rest/internal/http/routes.go`
    - `services/bff-rest/internal/http/middleware/jwt.go`
    - `services/bff-rest/internal/clients/grpc/{auth,notes}.go`
  - DoD: `/healthz` 200; защищённые ручки требуют токен.

- [ ] Swagger/OpenAPI
  - Файлы: `services/bff-rest/openapi/openapi.yaml`
  - DoD: UI дистрибутив (redoc/swagger-ui) отдает схему локально.

### Этап F — Сага

- [ ] Оркестратор
  - Файлы:
    - `services/saga-orchestrator/cmd/server/main.go`
    - `services/saga-orchestrator/internal/saga/signup.go`
    - `services/saga-orchestrator/internal/store/sagas_repo.go`
  - Переменные: `KAFKA_BROKERS`, `SAGA_TOPICS`, `DB_DSN`.
  - DoD: обработка `UserCreated` → создание welcome-note.

- [ ] Компенсации и таймауты
  - Реализация: контекст с deadline, ретраи с backoff, вызов `AuthService.DeleteUser`.
  - DoD: e2e тест с принудительной ошибкой в notes завершает компенсацией.

### Этап G — Наблюдаемость

- [ ] OTel
  - Команды:
    ```bash
    go get go.opentelemetry.io/otel/sdk@latest
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@latest
    ```
  - DoD: трассы видны в Jaeger; контекст передаётся через gRPC/Kafka.

- [ ] Prometheus
  - Команды:
    ```bash
    go get github.com/prometheus/client_golang/prometheus/promhttp@latest
    ```
  - DoD: `/metrics` отдаёт метрики во всех сервисах.

### Этап H — Docker/Compose

- [ ] Dockerfiles
  - Минимум:
    - `services/*/Dockerfile` (multi-stage: builder + runtime)
  - DoD: `docker build` успешен для каждого сервиса.

- [ ] docker-compose сервисы
  - Файл: `deploy/docker-compose/docker-compose.yaml` дополняется сервисами: auth, notes, bff, orchestrator, postgres*2, jaeger.
  - DoD: `docker compose up` поднимает весь стек; health-зелёные.

### Этап I — Kubernetes (dev)

- [ ] Helm/Kustomize
  - Файлы: `deploy/helm/notes-platform/` либо `deploy/k8s/{base,overlays/dev}`.
  - DoD: деплой dev namespace; `kubectl get pods` — все Running/Ready.

### Этап J — CI/CD

- [ ] GitHub Actions
  - Файл: `.github/workflows/ci.yaml` (этапы: lint, test, buf, buildx, compose-test).
  - DoD: все job-ы зелёные в PR; блокировка merge при провале.

