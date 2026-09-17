#!/usr/bin/env bash
# ==============================================================================
# Скрипт автоматической установки скомпилированного релиза Boyan на VPS Linux
# Запуск: sudo bash install.sh
# ==============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}   Next-Gen OPDS Suite («Боян») — Установка на VPS    ${NC}"
echo -e "${BLUE}======================================================${NC}"

# 1. Проверка прав root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Ошибка: Запустите установщик с правами root (sudo bash install.sh)${NC}"
    exit 1
fi

# 2. Создание пользователя boyan (если не создан)
if ! id -u boyan >/dev/null 2>&1; then
    useradd -r -s /bin/false -d /opt/boyan boyan
    echo -e "${GREEN}[+] Создан системный пользователь: boyan${NC}"
else
    echo -e "[*] Пользователь boyan уже существует"
fi

# 3. Создание файловой структуры
echo -e "[*] Создание рабочих директорий..."
mkdir -p /opt/boyan
mkdir -p /var/www/boyan/web-desktop
mkdir -p /var/www/boyan/web-mobile
mkdir -p /var/lib/boyan/data
mkdir -p /var/lib/boyan/library
mkdir -p /var/lib/boyan/import
mkdir -p /var/cache/boyan/covers

# 4. Копирование исполняемого файла
echo -e "[*] Установка бинарника бэкенда..."
cp boyan /opt/boyan/boyan
chmod +x /opt/boyan/boyan

# 5. Копирование или сохранение конфига
if [ ! -f /opt/boyan/config.yaml ]; then
    cp config.example.yaml /opt/boyan/config.yaml
    # Адаптируем пути под Linux VPS по умолчанию
    sed -i 's|path: "boyan.db"|path: "/var/lib/boyan/data/boyan.db"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|library_dir: "library"|library_dir: "/var/lib/boyan/library"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|watch_dir: "import"|watch_dir: "/var/lib/boyan/import"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|quarantine_dir: "quarantine"|quarantine_dir: "/var/lib/boyan/data/quarantine"|g' /opt/boyan/config.yaml 2>/dev/null || true
    echo -e "${GREEN}[+] Создан конфигурационный файл: /opt/boyan/config.yaml${NC}"
else
    echo -e "[*] Конфигурация /opt/boyan/config.yaml сохранена"
fi

# 6. Развертывание статики веб-интерфейсов
echo -e "[*] Развертывание статики веб-интерфейсов..."
if [ -d "web-desktop" ]; then
    rm -rf /var/www/boyan/web-desktop/*
    cp -r web-desktop/* /var/www/boyan/web-desktop/
    echo -e "${GREEN}[+] Десктопный фронтенд установлен в /var/www/boyan/web-desktop${NC}"
fi

if [ -d "web-mobile" ]; then
    rm -rf /var/www/boyan/web-mobile/*
    cp -r web-mobile/* /var/www/boyan/web-mobile/
    echo -e "${GREEN}[+] Мобильный PWA фронтенд установлен в /var/www/boyan/web-mobile${NC}"
fi

# 7. Выставление прав доступа
echo -e "[*] Настройка прав доступа к файлам..."
chown -R boyan:boyan /opt/boyan /var/lib/boyan /var/cache/boyan
chmod 750 /var/lib/boyan /var/cache/boyan
if id -u www-data >/dev/null 2>&1; then
    chown -R www-data:www-data /var/www/boyan
    chmod -R 755 /var/www/boyan
fi

# 8. Установка systemd службы
echo -e "[*] Регистрация systemd службы..."
cp boyan.service /etc/systemd/system/boyan.service
systemctl daemon-reload
systemctl enable boyan
systemctl restart boyan
echo -e "${GREEN}[+] Служба boyan запущена и добавлена в автозагрузку${NC}"

# 9. Настройка Nginx
if [ -d "/etc/nginx/sites-available" ] && [ -f "nginx-boyan.conf" ]; then
    cp nginx-boyan.conf /etc/nginx/sites-available/boyan
    if [ ! -f "/etc/nginx/sites-enabled/boyan" ]; then
        ln -s /etc/nginx/sites-available/boyan /etc/nginx/sites-enabled/boyan
    fi
    if nginx -t 2>/dev/null; then
        systemctl reload nginx
        echo -e "${GREEN}[+] Конфигурация Nginx применена (/etc/nginx/sites-available/boyan)${NC}"
    else
        echo -e "${RED}[!] Внимание: Проверьте server_name в /etc/nginx/sites-available/boyan и перезапустите nginx (systemctl reload nginx)${NC}"
    fi
fi

echo -e "\n${GREEN}======================================================${NC}"
echo -e "${GREEN}      Установка комплекса «Боян» успешно завершена!   ${NC}"
echo -e "${GREEN}======================================================${NC}"
echo -e "Проверка статуса:  systemctl status boyan"
echo -e "Просмотр логов:    journalctl -u boyan -f"
echo -e "Книги для импорта: /var/lib/boyan/import/"
echo -e "Конфигурация:      /opt/boyan/config.yaml\n"
