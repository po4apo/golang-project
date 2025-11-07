#!/bin/bash

###############################################################################
# Manual Deployment Script
# Используется для ручного деплоя без GitHub Actions
###############################################################################

set -e

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Показать помощь
show_help() {
    cat << EOF
${BLUE}Manual Deployment Script${NC}

Использование:
  ./manual-deploy.sh [OPTIONS]

Опции:
  -h, --host HOST          IP адрес или hostname сервера (обязательно)
  -u, --user USER          SSH пользователь (по умолчанию: root)
  -p, --path PATH          Путь для деплоя на сервере (по умолчанию: /opt/golang-project)
  -t, --tag TAG            Docker image tag (по умолчанию: latest)
  -r, --registry REGISTRY  Docker registry (по умолчанию: ghcr.io)
  --help                   Показать эту справку

Примеры:
  # Деплой на production сервер
  ./manual-deploy.sh -h 88.218.169.245 -u root -p /opt/golang-project

  # Деплой с конкретным тегом
  ./manual-deploy.sh -h 88.218.169.245 -t v1.2.3

Перед запуском убедитесь, что:
  1. У вас настроен SSH доступ к серверу
  2. На сервере установлен Docker и docker-compose
  3. Вы создали .env файл на сервере из .env.example
EOF
}

# Параметры по умолчанию
SERVER_HOST=""
SERVER_USER="root"
DEPLOY_PATH="/opt/golang-project"
IMAGE_TAG="latest"
REGISTRY="ghcr.io"
REPOSITORY=""

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            SERVER_HOST="$2"
            shift 2
            ;;
        -u|--user)
            SERVER_USER="$2"
            shift 2
            ;;
        -p|--path)
            DEPLOY_PATH="$2"
            shift 2
            ;;
        -t|--tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Неизвестный параметр: $1"
            show_help
            exit 1
            ;;
    esac
done

# Проверка обязательных параметров
if [ -z "$SERVER_HOST" ]; then
    log_error "Не указан хост сервера!"
    echo ""
    show_help
    exit 1
fi

# Определение repository из git remote
if [ -z "$REPOSITORY" ]; then
    GIT_REMOTE=$(git config --get remote.origin.url || echo "")
    if [[ $GIT_REMOTE =~ github.com[:/](.+)\.git ]]; then
        REPOSITORY="${BASH_REMATCH[1]}"
        log_info "Определен repository: $REPOSITORY"
    else
        log_error "Не удалось определить repository из git remote"
        exit 1
    fi
fi

# Получение текущего commit
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

log_info "=========================================="
log_info "  Manual Deployment"
log_info "=========================================="
log_info "Server: $SERVER_USER@$SERVER_HOST"
log_info "Path: $DEPLOY_PATH"
log_info "Registry: $REGISTRY"
log_info "Repository: $REPOSITORY"
log_info "Tag: $IMAGE_TAG"
log_info "Branch: $GIT_BRANCH"
log_info "Commit: $GIT_COMMIT"
log_info "=========================================="

# Подтверждение
read -p "Продолжить деплой? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warn "Деплой отменен"
    exit 0
fi

# Проверка SSH доступа
log_info "Проверка SSH доступа..."
if ! ssh -o ConnectTimeout=5 "$SERVER_USER@$SERVER_HOST" "echo 'SSH OK'" > /dev/null 2>&1; then
    log_error "Не удается подключиться к серверу по SSH"
    exit 1
fi
log_info "SSH доступ подтвержден"

# Создание deployment package
log_info "Подготовка файлов для деплоя..."
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

cp -r deploy/docker-compose/* "$TEMP_DIR/"
cp scripts/deploy.sh "$TEMP_DIR/"
chmod +x "$TEMP_DIR/deploy.sh"

# Создание манифеста
cat > "$TEMP_DIR/deploy-manifest.json" <<EOF
{
  "version": "$IMAGE_TAG",
  "commit": "$GIT_COMMIT",
  "branch": "$GIT_BRANCH",
  "deployed_by": "$(whoami)@$(hostname)",
  "deployed_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "deployment_type": "manual"
}
EOF

# Копирование файлов на сервер
log_info "Копирование файлов на сервер..."
ssh "$SERVER_USER@$SERVER_HOST" "mkdir -p $DEPLOY_PATH"
scp -r "$TEMP_DIR"/* "$SERVER_USER@$SERVER_HOST:$DEPLOY_PATH/"

# Проверка .env файла
log_info "Проверка конфигурации на сервере..."
if ! ssh "$SERVER_USER@$SERVER_HOST" "test -f $DEPLOY_PATH/.env"; then
    log_warn ".env файл не найден на сервере!"
    log_warn "Создайте .env файл из .env.example:"
    log_warn "  cd $DEPLOY_PATH && cp .env.example .env"
    log_warn "  nano .env  # заполните необходимые значения"
    exit 1
fi

# Запуск деплоя на сервере
log_info "Запуск деплоя на сервере..."
ssh "$SERVER_USER@$SERVER_HOST" << ENDSSH
    set -e
    cd $DEPLOY_PATH
    
    # Загружаем .env
    export \$(cat .env | grep -v '^#' | xargs)
    
    # Переопределяем переменные для этого деплоя
    export REGISTRY=$REGISTRY
    export REPOSITORY=$REPOSITORY
    export IMAGE_TAG=$IMAGE_TAG
    export GIT_COMMIT=$GIT_COMMIT
    export DEPLOYED_AT=\$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    # Запускаем deployment скрипт
    ./deploy.sh
ENDSSH

# Финальная проверка
log_info "Проверка работоспособности сервисов..."
sleep 5

if curl -f "http://$SERVER_HOST:8080/health" > /dev/null 2>&1; then
    log_info "✅ REST API работает!"
else
    log_warn "⚠️  REST API не отвечает, проверьте логи"
fi

log_info "=========================================="
log_info "  Деплой завершен!"
log_info "=========================================="
log_info "Проверьте приложение:"
log_info "  curl http://$SERVER_HOST:8080/health"
log_info ""
log_info "Логи:"
log_info "  ssh $SERVER_USER@$SERVER_HOST 'cd $DEPLOY_PATH && docker-compose -f docker-compose.production.yml logs -f'"

