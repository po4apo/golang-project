#!/bin/bash

###############################################################################
# Server Setup Script
# Подготовка сервера для деплоя приложения
###############################################################################

set -e

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

# Проверка прав root
if [ "$EUID" -ne 0 ]; then
    log_error "Этот скрипт нужно запускать с правами root"
    exit 1
fi

log_info "=========================================="
log_info "  Настройка сервера для Golang Project"
log_info "=========================================="

# Обновление системы
log_info "Обновление системы..."
apt-get update
apt-get upgrade -y

# Установка необходимых пакетов
log_info "Установка необходимых пакетов..."
apt-get install -y \
    curl \
    wget \
    git \
    vim \
    htop \
    net-tools \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release

# Установка Docker
log_info "Проверка Docker..."
if ! command -v docker &> /dev/null; then
    log_info "Установка Docker..."
    
    # Добавление Docker GPG ключа
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    
    # Добавление Docker репозитория
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # Установка Docker
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io
    
    # Запуск Docker
    systemctl start docker
    systemctl enable docker
    
    log_info "✅ Docker установлен"
else
    log_info "✅ Docker уже установлен: $(docker --version)"
fi

# Установка Docker Compose
log_info "Проверка Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    log_info "Установка Docker Compose..."
    
    DOCKER_COMPOSE_VERSION="v2.24.0"
    curl -L "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    
    log_info "✅ Docker Compose установлен"
else
    log_info "✅ Docker Compose уже установлен: $(docker-compose --version)"
fi

# Установка golang-migrate (опционально, для миграций)
log_info "Установка golang-migrate..."
if ! command -v migrate &> /dev/null; then
    MIGRATE_VERSION="v4.17.0"
    curl -L "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz" | tar xvz
    mv migrate /usr/local/bin/migrate
    chmod +x /usr/local/bin/migrate
    log_info "✅ golang-migrate установлен"
else
    log_info "✅ golang-migrate уже установлен"
fi

# Создание директории для приложения
DEPLOY_PATH="/opt/golang-project"
log_info "Создание директории $DEPLOY_PATH..."
mkdir -p "$DEPLOY_PATH"
mkdir -p "$DEPLOY_PATH/backups"

# Настройка firewall (UFW)
log_info "Настройка firewall..."
if command -v ufw &> /dev/null; then
    # Разрешаем SSH
    ufw allow 22/tcp
    
    # Разрешаем HTTP/HTTPS
    ufw allow 8080/tcp
    ufw allow 443/tcp
    
    # Разрешаем gRPC
    ufw allow 50051/tcp
    
    # Включаем firewall
    ufw --force enable
    
    log_info "✅ Firewall настроен"
else
    log_warn "UFW не установлен, пропускаем настройку firewall"
fi

# Настройка логирования
log_info "Настройка логирования Docker..."
cat > /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "5"
  }
}
EOF

systemctl restart docker

# Создание пользователя для деплоя (опционально)
DEPLOY_USER="deploy"
if ! id "$DEPLOY_USER" &>/dev/null; then
    log_info "Создание пользователя $DEPLOY_USER..."
    useradd -m -s /bin/bash "$DEPLOY_USER"
    usermod -aG docker "$DEPLOY_USER"
    
    # Настройка SSH для deploy пользователя
    mkdir -p /home/$DEPLOY_USER/.ssh
    chmod 700 /home/$DEPLOY_USER/.ssh
    touch /home/$DEPLOY_USER/.ssh/authorized_keys
    chmod 600 /home/$DEPLOY_USER/.ssh/authorized_keys
    chown -R $DEPLOY_USER:$DEPLOY_USER /home/$DEPLOY_USER/.ssh
    
    log_info "✅ Пользователь $DEPLOY_USER создан"
    log_warn "Добавьте SSH ключ в /home/$DEPLOY_USER/.ssh/authorized_keys"
else
    log_info "✅ Пользователь $DEPLOY_USER уже существует"
fi

# Создание systemd сервиса для автозапуска (опционально)
log_info "Создание systemd сервиса..."
cat > /etc/systemd/system/golang-project.service <<EOF
[Unit]
Description=Golang Project Application
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$DEPLOY_PATH
ExecStart=/usr/local/bin/docker-compose -f $DEPLOY_PATH/docker-compose.production.yml up -d
ExecStop=/usr/local/bin/docker-compose -f $DEPLOY_PATH/docker-compose.production.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
log_info "✅ Systemd сервис создан (не активирован)"

# Настройка мониторинга (базовая)
log_info "Установка базовых инструментов мониторинга..."
apt-get install -y iotop iftop ncdu

# Вывод финальной информации
log_info "=========================================="
log_info "  Настройка сервера завершена! ✅"
log_info "=========================================="
log_info ""
log_info "Следующие шаги:"
log_info "1. Скопируйте deployment файлы в $DEPLOY_PATH"
log_info "2. Создайте .env файл из .env.example:"
log_info "   cd $DEPLOY_PATH && cp .env.example .env"
log_info "3. Заполните переменные в .env файле"
log_info "4. Настройте GitHub Actions secrets:"
log_info "   - SSH_PRIVATE_KEY: SSH ключ для доступа"
log_info "   - SERVER_HOST: IP адрес сервера"
log_info "   - SERVER_USER: SSH пользователь (root или deploy)"
log_info "   - DEPLOY_PATH: $DEPLOY_PATH"
log_info "   - POSTGRES_PASSWORD: пароль БД"
log_info "   - JWT_SECRET_KEY: JWT ключ"
log_info ""
log_info "Полезные команды:"
log_info "  systemctl enable golang-project  # Автозапуск при загрузке"
log_info "  systemctl start golang-project   # Запуск приложения"
log_info "  docker ps                        # Список контейнеров"
log_info "  docker-compose logs -f           # Просмотр логов"
log_info ""
log_info "Установленные версии:"
docker --version
docker-compose --version
migrate -version 2>/dev/null || echo "migrate: не установлен"

