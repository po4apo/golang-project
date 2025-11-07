#!/bin/bash

# Скрипт для отката к предыдущей версии приложения
# Использование: ./rollback.sh [version]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Проверяем наличие docker-compose
if ! command -v docker-compose &> /dev/null; then
    log_error "docker-compose is not installed"
    exit 1
fi

# Директория с конфигурацией
COMPOSE_DIR="${COMPOSE_DIR:-/opt/golang-project/deploy/docker-compose}"
BACKUP_DIR="${BACKUP_DIR:-${COMPOSE_DIR}/backups}"

if [ ! -d "$COMPOSE_DIR" ]; then
    log_error "Compose directory not found: $COMPOSE_DIR"
    exit 1
fi

cd "$COMPOSE_DIR"

# Функция для создания бэкапа текущей конфигурации
backup_current() {
    log_info "Creating backup of current configuration..."
    
    mkdir -p "$BACKUP_DIR"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${BACKUP_DIR}/docker-compose.override.${timestamp}.yml"
    
    if [ -f "docker-compose.override.yml" ]; then
        cp docker-compose.override.yml "$backup_file"
        log_info "Backup created: $backup_file"
        echo "$backup_file"
    else
        log_warning "No override file found to backup"
        echo ""
    fi
}

# Функция для отката к конкретной версии
rollback_to_version() {
    local version=$1
    
    if [ -z "$version" ]; then
        log_error "Version not specified"
        exit 1
    fi
    
    log_info "Rolling back to version: $version"
    
    # Создаем новый override файл
    cat > docker-compose.override.yml << EOF
version: '3.8'
services:
  auth-service:
    image: ${REGISTRY}/${REPOSITORY}/auth-service:${version}
  
  rest-api:
    image: ${REGISTRY}/${REPOSITORY}/rest-api:${version}
EOF
    
    # Пулим образы
    log_info "Pulling images for version $version..."
    docker-compose -f docker-compose.yml -f docker-compose.override.yml pull
    
    # Перезапускаем сервисы
    log_info "Restarting services..."
    docker-compose -f docker-compose.yml -f docker-compose.override.yml up -d
    
    log_info "Waiting for services to start..."
    sleep 10
    
    # Проверяем health
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log_info "✅ Rollback successful! Services are healthy."
    else
        log_warning "⚠️ Services restarted but health check failed. Check logs."
    fi
}

# Функция для отката к последнему бэкапу
rollback_to_last_backup() {
    log_info "Finding last backup..."
    
    if [ ! -d "$BACKUP_DIR" ]; then
        log_error "No backups found in $BACKUP_DIR"
        exit 1
    fi
    
    local last_backup=$(ls -t "${BACKUP_DIR}"/docker-compose.override.*.yml 2>/dev/null | head -n 1)
    
    if [ -z "$last_backup" ]; then
        log_error "No backups found"
        exit 1
    fi
    
    log_info "Restoring from backup: $last_backup"
    
    cp "$last_backup" docker-compose.override.yml
    
    log_info "Restarting services..."
    docker-compose -f docker-compose.yml -f docker-compose.override.yml up -d
    
    log_info "Waiting for services to start..."
    sleep 10
    
    # Проверяем health
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log_info "✅ Rollback successful! Services are healthy."
    else
        log_warning "⚠️ Services restarted but health check failed. Check logs."
    fi
}

# Функция для показа доступных версий
list_versions() {
    log_info "Available versions from backups:"
    
    if [ ! -d "$BACKUP_DIR" ]; then
        log_warning "No backup directory found"
        return
    fi
    
    local backups=$(ls -t "${BACKUP_DIR}"/docker-compose.override.*.yml 2>/dev/null)
    
    if [ -z "$backups" ]; then
        log_warning "No backups found"
    else
        echo "$backups" | while read backup; do
            local timestamp=$(basename "$backup" | sed 's/docker-compose.override.\(.*\).yml/\1/')
            echo "  - $timestamp (file: $backup)"
        done
    fi
    
    echo ""
    log_info "Docker images in registry:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}" | grep -E "auth-service|rest-api" || log_warning "No images found locally"
}

# Main
case "${1:-help}" in
    version)
        if [ -z "$2" ]; then
            log_error "Please specify version to rollback to"
            echo "Usage: $0 version <version-tag>"
            exit 1
        fi
        backup_current
        rollback_to_version "$2"
        ;;
    last)
        backup_current
        rollback_to_last_backup
        ;;
    list)
        list_versions
        ;;
    help|*)
        echo "Usage: $0 {version|last|list}"
        echo ""
        echo "Commands:"
        echo "  version <tag>  - Rollback to specific version"
        echo "  last           - Rollback to last backup"
        echo "  list           - List available versions"
        echo ""
        echo "Examples:"
        echo "  $0 version main-abc12345"
        echo "  $0 last"
        echo "  $0 list"
        exit 1
        ;;
esac

