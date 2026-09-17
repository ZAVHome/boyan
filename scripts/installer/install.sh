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
    # Если комплекс уже установлен, берем домен из существующего конфига как дефолт
    EXISTING_DOMAIN=""
    if [ -f /opt/boyan/config.yaml ]; then
        EXISTING_DOMAIN=$(grep -E '^\s*base_url:' /opt/boyan/config.yaml | sed -E 's/.*https?:\/\/([^"/:]+).*/\1/' | head -n 1)
    fi
    DEFAULT_DOMAIN="${EXISTING_DOMAIN:-books.MYDOMAIN.COM}"

    if [ -t 0 ]; then
        echo -e "${BLUE}[?] Введите ваш субдомен для библиотеки Боян [${DEFAULT_DOMAIN}]:${NC}"
        read -r -p "Субдомен: " INPUT_DOMAIN
        DOMAIN="${INPUT_DOMAIN:-$DEFAULT_DOMAIN}"
    else
        DOMAIN="$DEFAULT_DOMAIN"
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
mkdir -p /var/lib/boyan/quarantine
mkdir -p /var/cache/boyan/covers

# Остановка работающей службы перед заменой файлов (предотвращение ошибки 'Text file busy')
if systemctl is-active --quiet boyan 2>/dev/null; then
    echo -e "[*] Остановка запущенной службы boyan для обновления..."
    systemctl stop boyan
fi

# 4. Копирование исполняемого файла
echo -e "[*] Установка бинарника бэкенда..."
# Используем безопасную замену с unlink (cp --remove-destination), исключающую ошибку ETXTBSY
cp --remove-destination boyan /opt/boyan/boyan 2>/dev/null || (cp boyan /opt/boyan/boyan.tmp && chmod +x /opt/boyan/boyan.tmp && mv -f /opt/boyan/boyan.tmp /opt/boyan/boyan)
chmod +x /opt/boyan/boyan

# 5. Копирование или сохранение конфига
if [ ! -f /opt/boyan/config.yaml ]; then
    cp config.example.yaml /opt/boyan/config.yaml

    # Генерация уникальных криптостойких секретов
    JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c 64)
    PG_PASSWORD=$(openssl rand -hex 16 2>/dev/null || tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c 32)

    # Адаптируем пути, секреты и URL под Linux VPS (FHS 3.0)
    sed -i 's|host: "0.0.0.0"|host: "127.0.0.1"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i "s|jwt_secret: .*|jwt_secret: \"${JWT_SECRET}\"|g" /opt/boyan/config.yaml 2>/dev/null || true
    sed -i "s|password: \"secretpassword\"|password: \"${PG_PASSWORD}\"|g" /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|path: "\./data/opds.db"|path: "/var/lib/boyan/data/boyan.db"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|path: "boyan.db"|path: "/var/lib/boyan/data/boyan.db"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|library_dir: "\./library"|library_dir: "/var/lib/boyan/library"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|watch_dir: "\./import"|watch_dir: "/var/lib/boyan/import"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i 's|cover_cache_dir: "\./data/cache/covers"|cover_cache_dir: "/var/cache/boyan/covers"|g' /opt/boyan/config.yaml 2>/dev/null || true
    sed -i "s|base_url: .*|base_url: \"https://${DOMAIN}\"|g" /opt/boyan/config.yaml 2>/dev/null || true
    sed -i "s|https://books.example.com|https://${DOMAIN}|g" /opt/boyan/config.yaml 2>/dev/null || true

    echo -e "${GREEN}[+] Создан конфигурационный файл: /opt/boyan/config.yaml (сгенерирован уникальный JWT_SECRET)${NC}"
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
chown -R boyan:boyan /opt/boyan /var/cache/boyan /var/lib/boyan/data /var/lib/boyan/quarantine /var/lib/boyan/import
chown boyan:boyan /var/lib/boyan /var/lib/boyan/library 2>/dev/null || true
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

# 9. Настройка SSL-сертификатов и Nginx (Strict HTTPS)
if [ -d "/etc/nginx/sites-available" ] && [ -f "nginx-boyan.conf" ]; then
    NGINX_TARGET="/etc/nginx/sites-available/boyan"
    NGINX_ENABLED="/etc/nginx/sites-enabled/boyan"
    mkdir -p /var/www/html/.well-known/acme-challenge
    chown -R www-data:www-data /var/www/html 2>/dev/null || true

    SSL_CERT=""
    SSL_KEY=""

    # 9.1. Проверка внешних переменных окружения (для автоматических установок)
    if [ -n "${SSL_CERT_FILE:-}" ] && [ -n "${SSL_KEY_FILE:-}" ] && [ -f "$SSL_CERT_FILE" ] && [ -f "$SSL_KEY_FILE" ]; then
        echo -e "${GREEN}[+] Использование SSL-сертификата из переменных окружения:${NC}"
        echo -e "    Cert: ${SSL_CERT_FILE}"
        echo -e "    Key:  ${SSL_KEY_FILE}"
        mkdir -p /etc/ssl/boyan
        cp "$SSL_CERT_FILE" /etc/ssl/boyan/fullchain.pem
        cp "$SSL_KEY_FILE" /etc/ssl/boyan/privkey.pem
        chmod 600 /etc/ssl/boyan/privkey.pem
        SSL_CERT="/etc/ssl/boyan/fullchain.pem"
        SSL_KEY="/etc/ssl/boyan/privkey.pem"
    fi

    # 9.2. Автоматическое обнаружение ранее или заранее выпущенных сертификатов
    if [ -z "$SSL_CERT" ]; then
        if [ -f "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ] && [ -f "/etc/letsencrypt/live/${DOMAIN}/privkey.pem" ]; then
            echo -e "${GREEN}[+] Обнаружен действующий SSL-сертификат Let's Encrypt: /etc/letsencrypt/live/${DOMAIN}/${NC}"
            SSL_CERT="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
            SSL_KEY="/etc/letsencrypt/live/${DOMAIN}/privkey.pem"
        elif [ -f "/etc/ssl/boyan/fullchain.pem" ] && [ -f "/etc/ssl/boyan/privkey.pem" ]; then
            echo -e "${GREEN}[+] Обнаружен заранее подготовленный SSL-сертификат: /etc/ssl/boyan/${NC}"
            SSL_CERT="/etc/ssl/boyan/fullchain.pem"
            SSL_KEY="/etc/ssl/boyan/privkey.pem"
        elif [ -f "/etc/ssl/certs/${DOMAIN}.crt" ] && [ -f "/etc/ssl/private/${DOMAIN}.key" ]; then
            echo -e "${GREEN}[+] Обнаружен SSL-сертификат: /etc/ssl/certs/${DOMAIN}.crt${NC}"
            SSL_CERT="/etc/ssl/certs/${DOMAIN}.crt"
            SSL_KEY="/etc/ssl/private/${DOMAIN}.key"
        elif [ -f "/etc/ssl/certs/${DOMAIN}.fullchain.pem" ] && [ -f "/etc/ssl/private/${DOMAIN}.privkey.pem" ]; then
            echo -e "${GREEN}[+] Обнаружен SSL-сертификат: /etc/ssl/certs/${DOMAIN}.fullchain.pem${NC}"
            SSL_CERT="/etc/ssl/certs/${DOMAIN}.fullchain.pem"
            SSL_KEY="/etc/ssl/private/${DOMAIN}.privkey.pem"
        fi
    fi

    # 9.3. Запрос пути к заранее выпущенному сертификату (в интерактивном режиме)
    if [ -z "$SSL_CERT" ] && [ -t 0 ]; then
        echo -e "\n${BLUE}[?] У вас уже есть заранее выпущенный SSL-сертификат (Wildcard, собственный CA или коммерческий)? [y/N]:${NC}"
        read -r -p "Использовать готовый сертификат? [y/N]: " HAS_PREISSUED_SSL
        if [[ "$HAS_PREISSUED_SSL" =~ ^[Yy]$ ]]; then
            read -r -p "Введите абсолютный путь к файлу сертификата (fullchain.pem / cert.crt): " USER_CERT
            read -r -p "Введите абсолютный путь к приватному ключу (privkey.pem / cert.key): " USER_KEY
            if [ -f "$USER_CERT" ] && [ -f "$USER_KEY" ]; then
                mkdir -p /etc/ssl/boyan
                cp "$USER_CERT" /etc/ssl/boyan/fullchain.pem
                cp "$USER_KEY" /etc/ssl/boyan/privkey.pem
                chmod 600 /etc/ssl/boyan/privkey.pem
                SSL_CERT="/etc/ssl/boyan/fullchain.pem"
                SSL_KEY="/etc/ssl/boyan/privkey.pem"
                echo -e "${GREEN}[+] Заранее выпущенный сертификат успешно скопирован в /etc/ssl/boyan/${NC}"
            else
                echo -e "${RED}[!] Указанные файлы не найдены. Будет предложен автоматический выпуск или fallback.${NC}"
            fi
        fi
    fi

    # 9.4. Выпуск через Certbot (HTTP-01 Webroot) при отсутствии сертификата
    if [ -z "$SSL_CERT" ]; then
        CAN_RUN_CERTBOT=false
        if [ "$DOMAIN" != "books.MYDOMAIN.COM" ] && [ "$DOMAIN" != "localhost" ] && [ "$DOMAIN" != "127.0.0.1" ]; then
            if command -v certbot >/dev/null 2>&1; then
                CAN_RUN_CERTBOT=true
            fi
        fi

        DO_CERTBOT=false
        if [ "$CAN_RUN_CERTBOT" = true ]; then
            if [ -t 0 ]; then
                read -r -p "Запустить Certbot для автоматического выпуска SSL сертификата на ${DOMAIN}? [Y/n]: " ASK_CERTBOT
                if [[ "${ASK_CERTBOT:-y}" =~ ^[Yy]$ ]]; then
                    DO_CERTBOT=true
                fi
            else
                DO_CERTBOT=true
            fi
        fi

        if [ "$DO_CERTBOT" = true ]; then
            echo -e "[*] Развертывание временной конфигурации Nginx (порт 80) для прохождения ACME-проверки Certbot..."
            cat << 'EOF_BOOTSTRAP' > "$NGINX_TARGET"
server {
    listen 80;
    listen [::]:80;
    server_name DOMAIN_PLACEHOLDER;

    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 200 "Boyan ACME bootstrap in progress...\n";
    }
}
EOF_BOOTSTRAP
            sed -i "s/DOMAIN_PLACEHOLDER/${DOMAIN}/g" "$NGINX_TARGET"
            [ ! -f "$NGINX_ENABLED" ] && ln -s "$NGINX_TARGET" "$NGINX_ENABLED"
            nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true

            echo -e "[*] Запуск Certbot в режиме webroot для домена ${DOMAIN}..."
            if certbot certonly --webroot -w /var/www/html -d "${DOMAIN}" --non-interactive --agree-tos --register-unsafely-without-email 2>/dev/null || certbot certonly --webroot -w /var/www/html -d "${DOMAIN}"; then
                SSL_CERT="/etc/letsencrypt/live/${DOMAIN}/fullchain.pem"
                SSL_KEY="/etc/letsencrypt/live/${DOMAIN}/privkey.pem"
                echo -e "${GREEN}[+] SSL-сертификат Let's Encrypt успешно получен!${NC}"
            else
                echo -e "${RED}[!] Certbot не смог получить сертификат (возможно, DNS-запись еще не обновилась).${NC}"
            fi
        fi

        # 9.5. Fallback: Генерация самоподписанного сертификата, если сертификат все еще отсутствует
        if [ -z "$SSL_CERT" ]; then
            echo -e "[*] Генерация самоподписанного SSL-сертификата для включения защищенного режима HTTPS..."
            mkdir -p /etc/ssl/boyan
            SSL_CERT="/etc/ssl/boyan/fullchain.pem"
            SSL_KEY="/etc/ssl/boyan/privkey.pem"
            openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
                -keyout "$SSL_KEY" \
                -out "$SSL_CERT" \
                -subj "/CN=${DOMAIN}" 2>/dev/null
            chmod 600 "$SSL_KEY"
            echo -e "${BLUE}[*] Создан рабочий сертификат в /etc/ssl/boyan/ (HTTPS активен)${NC}"
            if [ "$DOMAIN" != "books.MYDOMAIN.COM" ] && [ "$DOMAIN" != "localhost" ]; then
                echo -e "    ${RED}[!] Для замены на Let's Encrypt после обновления DNS выполните:${NC}"
                echo -e "    sudo certbot certonly --webroot -w /var/www/html -d ${DOMAIN}"
                echo -e "    sudo sed -i 's|/etc/ssl/boyan|/etc/letsencrypt/live/${DOMAIN}|g' ${NGINX_TARGET}"
                echo -e "    sudo systemctl reload nginx"
            fi
        fi
    fi

    # 9.6. Развертывание боевой конфигурации Nginx (Strict HTTPS)
    echo -e "[*] Развертывание боевой конфигурации Nginx (Strict HTTPS: порт 80 -> 301 редирект, порт 443 -> Boyan)..."
    cp nginx-boyan.conf "$NGINX_TARGET"
    sed -i "s/books.MYDOMAIN.COM/${DOMAIN}/g" "$NGINX_TARGET"
    sed -i "s|/etc/letsencrypt/live/${DOMAIN}/fullchain.pem|${SSL_CERT}|g" "$NGINX_TARGET"
    sed -i "s|/etc/letsencrypt/live/${DOMAIN}/privkey.pem|${SSL_KEY}|g" "$NGINX_TARGET"

    # Отключение конфликтующего стандартного хоста Ubuntu/Debian при наличии
    if [ -L "/etc/nginx/sites-enabled/default" ] && grep -q "Welcome to nginx" /etc/nginx/sites-available/default 2>/dev/null; then
        rm -f /etc/nginx/sites-enabled/default
    fi

    if [ ! -f "$NGINX_ENABLED" ]; then
        ln -s "$NGINX_TARGET" "$NGINX_ENABLED"
    fi

    if nginx -t 2>/dev/null; then
        systemctl reload nginx
        echo -e "${GREEN}[+] Конфигурация Nginx успешно применена и запущена!${NC}"
    else
        echo -e "${RED}[!] Ошибка тестирования конфигурации Nginx. Проверьте вывод: sudo nginx -t${NC}"
    fi
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
