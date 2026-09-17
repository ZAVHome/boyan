#!/usr/bin/env bash
# ==============================================================================
# Скрипт сборки автономного релизного пакета для Linux VPS (Bash / Linux / macOS)
# Результат: dist/boyan-linux-amd64.tar.gz
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
STAGING_DIR="${DIST_DIR}/boyan-linux-amd64"
ARCHIVE_FILE="${DIST_DIR}/boyan-linux-amd64.tar.gz"

echo "==> Очистка предыдущих сборок..."
rm -rf "${STAGING_DIR}" "${ARCHIVE_FILE}"
mkdir -p "${STAGING_DIR}"

echo "==> 1. Сборка Go бэкенда для Linux (amd64, без CGO)..."
(
    cd "${ROOT_DIR}/backend"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${STAGING_DIR}/boyan" ./cmd/server
)

echo "==> 2. Сборка десктопного фронтенда (web-desktop)..."
(
    cd "${ROOT_DIR}/frontends/web-desktop"
    npm run build
    cp -r dist "${STAGING_DIR}/web-desktop"
)

echo "==> 3. Сборка мобильного фронтенда (web-mobile)..."
(
    cd "${ROOT_DIR}/frontends/web-mobile"
    npm run build
    cp -r dist "${STAGING_DIR}/web-mobile"
)

echo "==> 4. Копирование конфигураций и инсталлятора..."
cp "${ROOT_DIR}/config.example.yaml" "${STAGING_DIR}/config.example.yaml"
cp "${ROOT_DIR}/scripts/systemd/boyan.service" "${STAGING_DIR}/boyan.service"
cp "${ROOT_DIR}/scripts/nginx/boyan.conf" "${STAGING_DIR}/nginx-boyan.conf"
cp "${ROOT_DIR}/scripts/installer/install.sh" "${STAGING_DIR}/install.sh"
chmod +x "${STAGING_DIR}/install.sh"

echo "==> 5. Упаковка в tar.gz архив..."
(
    cd "${DIST_DIR}"
    tar -czf "boyan-linux-amd64.tar.gz" -C "${STAGING_DIR}" .
)

echo -e "\nУспешно сформирован релизный архив для VPS:"
echo "Файл: ${ARCHIVE_FILE}"
echo "Для установки на VPS скопируйте архив и запустите:"
echo "  scp ${ARCHIVE_FILE} root@your-vps:/tmp/"
echo "  ssh root@your-vps"
echo "  mkdir /tmp/boyan-pkg && tar -xzf /tmp/boyan-linux-amd64.tar.gz -C /tmp/boyan-pkg"
echo "  cd /tmp/boyan-pkg && sudo bash install.sh"
