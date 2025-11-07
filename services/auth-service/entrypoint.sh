#!/bin/sh
set -e

echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h postgres-auth -U authuser -d authdb > /dev/null 2>&1; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done

echo "PostgreSQL is up - applying migrations"

# Устанавливаем migrate если его нет
if ! command -v migrate &> /dev/null; then
    echo "Installing migrate..."
    wget -qO- https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
    mv migrate /usr/local/bin/migrate
    chmod +x /usr/local/bin/migrate
fi

# Применяем миграции
migrate -database "${DB_DSN}" -path ./migrations up

echo "Migrations applied successfully"
echo "Starting auth-service..."

# Запускаем сервис
exec ./auth-service





