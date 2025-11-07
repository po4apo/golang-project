#!/bin/bash

# Скрипт для проверки состояния сервисов
# Использование: ./health-check.sh [service]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
REST_API_URL="${REST_API_URL:-http://localhost:8080}"
MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-2}"

# Функции для логирования
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Проверка REST API
check_rest_api() {
    log_info "Checking REST API health..."
    
    local retry=0
    while [ $retry -lt $MAX_RETRIES ]; do
        if curl -f -s "${REST_API_URL}/health" > /dev/null 2>&1; then
            log_success "REST API is healthy! ✅"
            return 0
        fi
        
        retry=$((retry + 1))
        if [ $retry -lt $MAX_RETRIES ]; then
            log_warning "Attempt ${retry}/${MAX_RETRIES}: REST API not ready yet..."
            sleep $RETRY_INTERVAL
        fi
    done
    
    log_error "REST API health check failed after ${MAX_RETRIES} attempts ❌"
    return 1
}

# Проверка регистрации пользователя
check_signup() {
    log_info "Testing user registration..."
    
    local test_email="healthcheck_$(date +%s)@example.com"
    local test_password="HealthCheck123!"
    
    local response=$(curl -s -w "\n%{http_code}" -X POST "${REST_API_URL}/api/v1/auth/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${test_email}\",\"password\":\"${test_password}\"}")
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "409" ]; then
        log_success "User registration endpoint is working! ✅"
        return 0
    else
        log_error "User registration failed with HTTP ${http_code}: ${body} ❌"
        return 1
    fi
}

# Проверка входа пользователя
check_signin() {
    log_info "Testing user sign-in..."
    
    # Сначала создаем пользователя
    local test_email="healthcheck_signin_$(date +%s)@example.com"
    local test_password="HealthCheck123!"
    
    curl -s -X POST "${REST_API_URL}/api/v1/auth/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${test_email}\",\"password\":\"${test_password}\"}" > /dev/null
    
    # Пытаемся войти
    local response=$(curl -s -w "\n%{http_code}" -X POST "${REST_API_URL}/api/v1/auth/signin" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${test_email}\",\"password\":\"${test_password}\"}")
    
    local http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ]; then
        log_success "User sign-in endpoint is working! ✅"
        return 0
    else
        log_error "User sign-in failed with HTTP ${http_code} ❌"
        return 1
    fi
}

# Проверка Docker контейнеров
check_docker_containers() {
    log_info "Checking Docker containers..."
    
    local services=("auth-service" "rest-api" "postgres-auth")
    local all_healthy=true
    
    for service in "${services[@]}"; do
        if docker ps --filter "name=${service}" --filter "status=running" | grep -q "${service}"; then
            log_success "Container ${service} is running ✅"
        else
            log_error "Container ${service} is not running ❌"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        return 0
    else
        return 1
    fi
}

# Полная проверка системы
full_check() {
    log_info "==================== Full Health Check ===================="
    
    local failed=0
    
    # Проверяем Docker контейнеры
    if ! check_docker_containers; then
        failed=$((failed + 1))
    fi
    
    echo ""
    
    # Проверяем REST API
    if ! check_rest_api; then
        failed=$((failed + 1))
    fi
    
    echo ""
    
    # Проверяем регистрацию
    if ! check_signup; then
        failed=$((failed + 1))
    fi
    
    echo ""
    
    # Проверяем вход
    if ! check_signin; then
        failed=$((failed + 1))
    fi
    
    echo ""
    log_info "==========================================================="
    
    if [ $failed -eq 0 ]; then
        log_success "All checks passed! System is healthy! 🎉"
        return 0
    else
        log_error "${failed} check(s) failed! ❌"
        return 1
    fi
}

# Вывод статистики
show_stats() {
    log_info "==================== System Statistics ===================="
    
    echo ""
    log_info "Docker Containers:"
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "auth-service|rest-api|postgres"
    
    echo ""
    log_info "Resource Usage:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep -E "NAME|auth-service|rest-api|postgres"
    
    echo ""
    log_info "Recent Logs (last 10 lines):"
    echo "--- REST API ---"
    docker logs rest-api --tail 10 2>&1 || log_warning "Could not fetch REST API logs"
    
    echo ""
    log_info "==========================================================="
}

# Main
case "${1:-full}" in
    rest-api)
        check_rest_api
        ;;
    signup)
        check_signup
        ;;
    signin)
        check_signin
        ;;
    docker)
        check_docker_containers
        ;;
    stats)
        show_stats
        ;;
    full)
        full_check
        ;;
    *)
        echo "Usage: $0 {rest-api|signup|signin|docker|stats|full}"
        echo ""
        echo "Commands:"
        echo "  rest-api  - Check REST API health endpoint"
        echo "  signup    - Test user registration"
        echo "  signin    - Test user sign-in"
        echo "  docker    - Check Docker container status"
        echo "  stats     - Show system statistics"
        echo "  full      - Run all checks (default)"
        exit 1
        ;;
esac

