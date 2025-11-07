#!/bin/bash

###############################################################################
# Deployment Script for Golang Project
# Используется GitHub Actions для автоматического деплоя
###############################################################################

set -e  # Выход при ошибке
set -u  # Выход при использовании неопределенных переменных

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функции для логирования
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверка переменных окружения
check_env_vars() {
    log_info "Проверка переменных окружения..."
    
    local required_vars=(
        "REGISTRY"
        "REPOSITORY"
        "IMAGE_TAG"
        "POSTGRES_PASSWORD"
        "JWT_SECRET_KEY"
    )
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var:-}" ]; then
            log_error "Переменная окружения $var не установлена!"
            exit 1
        fi
    done
    
    log_info "Все необходимые переменные окружения установлены"
}

# Создание .env файла
create_env_file() {
    log_info "Создание .env файла..."
    
    cat > .env <<EOF
# Docker Registry Configuration
REGISTRY=${REGISTRY}
REPOSITORY=${REPOSITORY}
IMAGE_TAG=${IMAGE_TAG}

# Database Configuration
POSTGRES_DB=authdb
POSTGRES_USER=authuser
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_PORT=5432

# Services Configuration
AUTH_SERVICE_PORT=50051
REST_API_PORT=8080

# JWT Configuration
JWT_SECRET_KEY=${JWT_SECRET_KEY}

# Logging
LOG_LEVEL=${LOG_LEVEL:-info}

# Deployment Metadata
GIT_COMMIT=${GIT_COMMIT:-unknown}
DEPLOYED_AT=${DEPLOYED_AT:-unknown}
EOF
    
    log_info ".env файл создан"
}

# Backup текущей конфигурации
backup_current_state() {
    log_info "Создание backup текущего состояния..."
    
    local backup_dir="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    # Backup .env файла
    if [ -f .env ]; then
        cp .env "$backup_dir/.env.backup"
    fi
    
    # Backup docker-compose state
    docker-compose -f docker-compose.production.yml ps > "$backup_dir/containers_state.txt" 2>&1 || true
    
    log_info "Backup сохранен в $backup_dir"
}

# Login в Docker Registry
docker_login() {
    log_info "Вход в Docker Registry..."
    
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        echo "${GITHUB_TOKEN}" | docker login ${REGISTRY} -u ${GITHUB_ACTOR:-github} --password-stdin
        log_info "Успешный вход в Registry"
    else
        log_warn "GITHUB_TOKEN не установлен, пропускаем docker login"
    fi
}

# Pull новых образов
pull_images() {
    log_info "Загрузка новых Docker образов..."
    
    docker-compose -f docker-compose.production.yml pull
    
    log_info "Образы успешно загружены"
}

# Проверка состояния базы данных
check_database() {
    log_info "Проверка состояния базы данных..."
    
    # Проверяем, запущена ли БД
    if docker-compose -f docker-compose.production.yml ps postgres-auth | grep -q "Up"; then
        log_info "База данных уже запущена"
        return 0
    fi
    
    log_warn "База данных не запущена, запускаем..."
    docker-compose -f docker-compose.production.yml up -d postgres-auth
    
    # Ждем готовности БД
    log_info "Ожидание готовности базы данных..."
    for i in {1..30}; do
        if docker-compose -f docker-compose.production.yml exec -T postgres-auth pg_isready -U authuser > /dev/null 2>&1; then
            log_info "База данных готова"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    
    log_error "База данных не готова после 60 секунд ожидания"
    return 1
}

# Запуск миграций
run_migrations() {
    log_info "Запуск миграций базы данных..."
    
    # Проверяем наличие migrate
    if ! command -v migrate &> /dev/null; then
        log_warn "migrate не установлен, пропускаем миграции"
        log_warn "Установите migrate: https://github.com/golang-migrate/migrate"
        return 0
    fi
    
    # Запускаем миграции для auth-service
    local db_dsn="postgres://authuser:${POSTGRES_PASSWORD}@localhost:5432/authdb?sslmode=disable"
    
    if [ -d "../services/auth-service/migrations" ]; then
        migrate -database "$db_dsn" -path ../services/auth-service/migrations up || {
            log_warn "Миграции не удалось применить или уже применены"
        }
        log_info "Миграции завершены"
    else
        log_warn "Директория миграций не найдена"
    fi
}

# Graceful shutdown старых контейнеров
graceful_shutdown() {
    log_info "Остановка старых контейнеров..."
    
    # Даем контейнерам 30 секунд на graceful shutdown
    docker-compose -f docker-compose.production.yml stop -t 30 auth-service rest-api || true
    
    log_info "Контейнеры остановлены"
}

# Запуск новых контейнеров
start_services() {
    log_info "Запуск новых контейнеров..."
    
    # Запускаем с новыми образами
    docker-compose -f docker-compose.production.yml up -d auth-service rest-api
    
    log_info "Контейнеры запущены"
}

# Health check сервисов
health_check() {
    log_info "Проверка health сервисов..."
    
    local max_attempts=30
    local attempt=0
    
    # Проверка REST API
    log_info "Проверка REST API..."
    while [ $attempt -lt $max_attempts ]; do
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            log_info "✅ REST API работает корректно"
            break
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "REST API не отвечает после ${max_attempts} попыток"
        return 1
    fi
    
    log_info "Все сервисы работают корректно"
    return 0
}

# Cleanup старых образов
cleanup_old_images() {
    log_info "Очистка старых Docker образов..."
    
    # Удаляем dangling images
    docker image prune -f || true
    
    log_info "Очистка завершена"
}

# Отправка уведомления о деплое
send_deployment_notification() {
    log_info "Деплой завершен успешно!"
    
    # Сохраняем информацию о деплое
    cat > deployment_info.txt <<EOF
Deployment Information
=====================
Version: ${IMAGE_TAG}
Git Commit: ${GIT_COMMIT:-unknown}
Deployed At: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Registry: ${REGISTRY}
Repository: ${REPOSITORY}

Services Deployed:
- auth-service:${IMAGE_TAG}
- rest-api:${IMAGE_TAG}

Status: SUCCESS
EOF
    
    cat deployment_info.txt
}

# Rollback в случае неудачи
rollback() {
    log_error "Деплой не удался! Выполняется rollback..."
    
    # Находим последний успешный backup
    local latest_backup=$(ls -t backups/ | head -1)
    
    if [ -n "$latest_backup" ] && [ -f "backups/$latest_backup/.env.backup" ]; then
        log_info "Восстановление из backup: $latest_backup"
        cp "backups/$latest_backup/.env.backup" .env
        docker-compose -f docker-compose.production.yml up -d --force-recreate
        log_warn "Rollback выполнен"
    else
        log_error "Backup не найден, rollback невозможен"
    fi
    
    exit 1
}

# Главная функция
main() {
    log_info "=========================================="
    log_info "  Запуск деплоя приложения"
    log_info "=========================================="
    
    # Устанавливаем trap для rollback при ошибке
    trap rollback ERR
    
    check_env_vars
    backup_current_state
    create_env_file
    docker_login
    pull_images
    check_database
    run_migrations
    graceful_shutdown
    start_services
    
    # Даем сервисам время на запуск
    sleep 10
    
    if health_check; then
        cleanup_old_images
        send_deployment_notification
        log_info "=========================================="
        log_info "  Деплой завершен успешно! ✅"
        log_info "=========================================="
        exit 0
    else
        rollback
    fi
}

# Запуск
main "$@"
