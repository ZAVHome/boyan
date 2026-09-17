#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "===> Deploying Boyan (Next-Gen OPDS Suite) to WSL2 systemd..."

# Проверяем наличие скомпилированного бинарника под Linux
BINARY_SRC="${PROJECT_ROOT}/bin/boyan-linux"
if [[ ! -f "${BINARY_SRC}" ]]; then
    echo "ERROR: ${BINARY_SRC} not found!"
    echo "Please build the Linux binary first using: scripts/build-wsl.ps1 or 'make build-linux'"
    exit 1
fi

INSTALL_DIR="/opt/boyan"
echo "===> Preparing installation directory: ${INSTALL_DIR}..."
sudo mkdir -p "${INSTALL_DIR}"
sudo mkdir -p "${INSTALL_DIR}/data"
sudo mkdir -p "${INSTALL_DIR}/data/cache/covers"
sudo mkdir -p "${INSTALL_DIR}/library"
sudo mkdir -p "${INSTALL_DIR}/import"

# Останавливаем сервис, если он был запущен
if systemctl is-active --quiet boyan 2>/dev/null; then
    echo "===> Stopping currently running boyan service..."
    sudo systemctl stop boyan
fi

echo "===> Copying binary..."
sudo cp "${BINARY_SRC}" "${INSTALL_DIR}/boyan"
sudo chmod +x "${INSTALL_DIR}/boyan"

# Копируем конфигурацию, если еще не создана
if [[ ! -f "${INSTALL_DIR}/config.yaml" ]]; then
    echo "===> Initializing config.yaml in ${INSTALL_DIR}..."
    if [[ -f "${PROJECT_ROOT}/config.yaml" ]]; then
        sudo cp "${PROJECT_ROOT}/config.yaml" "${INSTALL_DIR}/config.yaml"
    else
        sudo cp "${PROJECT_ROOT}/config.example.yaml" "${INSTALL_DIR}/config.yaml"
    fi
fi

echo "===> Installing systemd service..."
sudo cp "${SCRIPT_DIR}/boyan.service" /etc/systemd/system/boyan.service
sudo systemctl daemon-reload
sudo systemctl enable boyan
sudo systemctl restart boyan

sleep 1

if systemctl is-active --quiet boyan; then
    echo "===> SUCCESS: Boyan service is ACTIVE and running in WSL2!"
    echo "     Health check: curl http://localhost:8080/health"
    echo "     API ping:     curl http://localhost:8080/api/v1/ping"
else
    echo "===> WARNING: Service failed to start. Journal logs:"
    sudo journalctl -u boyan -n 20 --no-pager
    exit 1
fi
