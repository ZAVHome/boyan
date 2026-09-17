#!/usr/bin/env bash
# ==============================================================================
# Скрипт автоматической установки скомпилированного релиза Боян на VPS Linux
# Запуск: sudo bash install.sh [books.MYDOMAIN.COM]
# ==============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}        Комплекс «Боян» — Установка на VPS            ${NC}"
echo -e "${BLUE}======================================================${NC}"

# 1. Проверка прав root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Ошибка: Запустите установщик с правами root (sudo bash install.sh [домен])${NC}"
    exit 1
fi

# Получение доменного имени / субдомена (например books.MYDOMAIN.COM)
DOMAIN="${1:-}"
if [ -z "$DOMAIN" ]; then
    if [ -t 0 ]; then
        echo -e "${BLUE}[?] Введите ваш субдомен для библиотеки Боян (например books.MYDOMAIN.COM):${NC}"
        read -r -p "Субдомен [books.MYDOMAIN.COM]: " INPUT_DOMAIN
        DOMAIN="${INPUT_DOMAIN:-books.MYDOMAIN.COM}"
    else
        DOMAIN="books.MYDOMAIN.COM"
    fi
fi
echo -e "[*] Настройка для субдомена: ${GREEN}${DOMAIN}${NC}"

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
    # Адаптируем пути и URL под Linux VPS
    sed -i 's|path: "boyan.db"|path: "/var/lib/boyan/data/boyan.db"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|library_dir: "library"|library_dir: "/var/lib/boyan/library"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|watch_dir: "import"|watch_dir: "/var/lib/boyan/import"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|quarantine_dir: "quarantine"|quarantine_dir: "/var/lib/boyan/data/quarantine"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i "s|base_url: .*|base_url: \"https://${DOMAIN}\"|g" /opt/boyan/config.yaml 2>/dev/null || true
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
    echo -e "[*] Настройка виртуального хоста Nginx для ${DOMAIN}..."
    cp nginx-boyan.conf /etc/nginx/sites-available/boyan
    sed -i "s/server_name books.MYDOMAIN.COM;/server_name ${DOMAIN};/g" /etc/nginx/sites-available/boyan
    sed -i "s/server_name your-domain.com;/server_name ${DOMAIN};/g" /etc/nginx/sites-available/boyan
    
    if [ ! -f "/etc/nginx/sites-enabled/boyan" ]; then
        ln -s /etc/nginx/sites-available/boyan /etc/nginx/sites-enabled/boyan
    fi
    if nginx -t 2>/dev/null; then
        systemctl reload nginx
        echo -e "${GREEN}[+] Конфигурация Nginx применена (/etc/nginx/sites-available/boyan)${NC}"
    else
        echo -e "${RED}[!] Внимание: Ошибка конфигурации Nginx. Проверьте /etc/nginx/sites-available/boyan${NC}"
    fi
fi

# 10. Настройка SSL-сертификата Let's Encrypt
if [ "$DOMAIN" != "books.MYDOMAIN.COM" ] && [ "$DOMAIN" != "localhost" ] && [ "$DOMAIN" != "127.0.0.1" ]; then
    echo -e "\n${BLUE}[*] Проверка настройки SSL-сертификата Let's Encrypt...${NC}"
    if command -v certbot >/dev/null 2>&1; then
        ISSUE_SSL="y"
        if [ -t 0 ]; then
            read -r -p "Запустить Certbot для автоматического выпуска SSL сертификата на ${DOMAIN}? [Y/n]: " ASK_SSL
            ISSUE_SSL="${ASK_SSL:-y}"
        fi
        if [[ "$ISSUE_SSL" =~ ^[Yy]$ ]]; then
            echo -e "[*] Запуск certbot --nginx -d ${DOMAIN}..."
            if certbot --nginx -d "${DOMAIN}" --redirect; then
                echo -e "${GREEN}[+] SSL-сертификат Let's Encrypt успешно выпущен и активирован!${NC}"
            else
                echo -e "${RED}[!] Предупреждение: Certbot не смог завершить процедуру.${NC}"
                echo -e "    Убедитесь, что DNS A-запись для субдомена ${DOMAIN} указывает на IP этого сервера."
                echo -e "    Повторный запуск: sudo certbot --nginx -d ${DOMAIN}"
            fi
        fi
    else
        echo -e "${BLUE}[*] Certbot не установлен.${NC} Для включения HTTPS и PWA выполните:"
        echo -e "    sudo apt install -y certbot python3-certbot-nginx"
        echo -e "    sudo certbot --nginx -d ${DOMAIN}"
    fi
else
    echo -e "\n[*] Для выпуска реального сертификата Let's Encrypt укажите рабочий субдомен и выполните:"
    echo -e "    sudo certbot --nginx -d ВАШ_СУБДОМЕН"
fi

echo -e "\n${GREEN}======================================================${NC}"
echo -e "${GREEN}      Установка комплекса «Боян» успешно завершена!   ${NC}"
echo -e "${GREEN}======================================================${NC}"
echo -e "🌐 Веб-интерфейс (Desktop): https://${DOMAIN}/"
echo -e "📱 Мобильное PWA (Mobile):  https://${DOMAIN}/m/"
echo -e "📚 OPDS каталог v1.2:       https://${DOMAIN}/opds/v1/feed.xml"
echo -e "📖 OPDS каталог v2.0:       https://${DOMAIN}/opds/v2/catalog.json"
echo -e "🔌 API документация:        https://${DOMAIN}/api/v1/docs/index.html"
echo -e "------------------------------------------------------"
echo -e "Проверка статуса службы:   systemctl status boyan"
echo -e "Просмотр логов:            journalctl -u boyan -f"
echo -e "Каталог импорта книг:      /var/lib/boyan/import/"
echo -e "Конфигурационный файл:     /opt/boyan/config.yaml\n"
