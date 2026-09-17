# Руководство по установке Next-Gen OPDS Suite («Боян») на Linux VPS

Это руководство подробно описывает процесс развертывания программного комплекса **«Боян»** на виртуальных выделенных серверах (VPS / VDS) под управлением Linux (**Ubuntu 22.04 / 24.04 LTS**, **Debian 11 / 12**, **Rocky Linux 9**).

---

## 💡 Философия развертывания: Zero-Build на боевом сервере

Традиционная сборка проектов прямо на сервере (установка компилятора Go, Node.js, npm, скачивание сотен мегабайт зависимостей) на недорогих VPS приводит к перерасходу оперативной памяти, рискам безопасности и замусориванию системы.

Архитектура комплекса «Боян» изначально спроектирована для **внешней сборки**:

1. **Бэкенд на Go** компилируется в один независимый статический бинарник без CGO (`CGO_ENABLED=0`), который не требует внешних библиотек и потребляет всего **~5 МБ RAM**.
2. **Фронтенды** (десктопный SPA и мобильный PWA) собираются во внешнем окружении в готовые оптимизированные статические файлы HTML/CSS/JS.
3. На VPS доставляется компактный готовый архив (~11–12 МБ), который сразу разворачивается и управляется системной службой `systemd` и веб-сервером `Nginx`.

---

## 🖥️ Системные требования VPS

| Характеристика | Минимальные требования | Рекомендуемые требования |
| :--- | :--- | :--- |
| **Процессор (vCPU)** | 1 ядро | 1–2 ядра |
| **Оперативная память** | 512 МБ RAM | 1 ГБ RAM |
| **Дисковое пространство** | 2 ГБ + размер книг | 10 ГБ + размер книг (SSD/NVMe) |
| **Операционная система** | Ubuntu 22.04+, Debian 11+ | Ubuntu 24.04 LTS |
| **Сеть** | Публичный IPv4 / IPv6 адрес | Доменное имя (для HTTPS) |

---

## Способ 1. Установка готового релиза в один клик (Рекомендуемый)

### Шаг 1. Формирование релизного архива на машине разработчика

На вашем рабочем компьютере (где есть исходный код, Go и Node.js) выполните скрипт сборки пакета:

- **В Windows (PowerShell):**

  ```powershell
  powershell -ExecutionPolicy Bypass -File scripts/package-release.ps1
  ```

- **В Linux / macOS / WSL2 (Bash):**

  ```bash
  bash scripts/package-release.sh
  ```

В результате в папке `dist/` будет создан готовый автономный архив:  
`dist/boyan-linux-amd64.tar.gz` (размером всего ~11.6 МБ).

*(Либо скачайте готовый архив `boyan-linux-amd64.tar.gz` со страницы [GitHub Releases](https://github.com/ZAVHome/boyan/releases)).*

---

### Шаг 2. Копирование архива на VPS

Передайте архив на ваш сервер через `scp`:

```bash
scp dist/boyan-linux-amd64.tar.gz root@IP_ВАШЕГО_СЕРВЕРА:/tmp/
```

---

### Шаг 3. Запуск установки на VPS

Подключитесь к серверу по SSH и запустите автоматическую установку:

```bash
ssh root@IP_ВАШЕГО_СЕРВЕРА

# Создаем временный каталог и распаковываем архив
mkdir -p /tmp/boyan-pkg
tar -xzf /tmp/boyan-linux-amd64.tar.gz -C /tmp/boyan-pkg
cd /tmp/boyan-pkg

# Запуск инсталлятора
sudo bash install.sh
```

**Что делает автоматический инсталлятор `install.sh`:**

1. Создает изолированного системного пользователя `boyan` без прав входа в шелл.
2. Создает структуру каталогов:
   - `/opt/boyan` — бинарник сервера и конфигурация `config.yaml`.
   - `/var/lib/boyan/data` — база данных SQLite 3 в режиме WAL.
   - `/var/lib/boyan/library` — постоянное хранилище книг.
   - `/var/lib/boyan/import` — папка мониторинга для автоматического импорта новых книг.
   - `/var/cache/boyan/covers` — LRU-кэш обложек.
   - `/var/www/boyan/web-desktop` — статика десктопного фронтенда.
   - `/var/www/boyan/web-mobile` — статика мобильного PWA-ридера.
3. Устанавливает и запускает службу `systemd` (`boyan.service`).
4. Подключает готовый конфигурационный файл Nginx (`/etc/nginx/sites-available/boyan`).

---

## Способ 2. Ручная пошаговая установка (Step-by-Step)

Если вы хотите выполнить каждый шаг вручную или настроить нестандартные пути, следуйте этой инструкции.

### 1. Подготовка сервера и установка базовых пакетов

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y nginx curl certbot python3-certbot-nginx ufw
```

### 2. Создание системного пользователя и директорий

В целях безопасности сервис не должен работать из-под учетной записи `root`.

```bash
# Создание пользователя без домашней директории и оболочки входа
sudo useradd -r -s /bin/false -d /opt/boyan boyan

# Создание структуры каталогов
sudo mkdir -p /opt/boyan
sudo mkdir -p /var/www/boyan/web-desktop
sudo mkdir -p /var/www/boyan/web-mobile
sudo mkdir -p /var/lib/boyan/data
sudo mkdir -p /var/lib/boyan/library
sudo mkdir -p /var/lib/boyan/import
sudo mkdir -p /var/cache/boyan/covers
```

### 3. Размещение бинарника и конфигурации

Скопируйте скомпилированный бинарник `boyan` и конфигурационный файл `config.example.yaml`:

```bash
sudo cp boyan /opt/boyan/boyan
sudo chmod +x /opt/boyan/boyan

sudo cp config.example.yaml /opt/boyan/config.yaml
```

Отредактируйте `/opt/boyan/config.yaml`, указав пути для Linux:

```yaml
server:
  host: "127.0.0.1"    # Слушать только локальный интерфейс (Nginx проксирует запросы)
  port: 8080
  jwt_secret: "УКАЖИТЕ_СЛУЧАЙНУЮ_СЕКРЕТНУЮ_СТРОКУ_ИЗ_32_СИМВОЛОВ"

database:
  driver: "sqlite"
  path: "/var/lib/boyan/data/boyan.db"

storage:
  library_dir: "/var/lib/boyan/library"
  watch_dir: "/var/lib/boyan/import"
  quarantine_dir: "/var/lib/boyan/data/quarantine"

covers:
  cache_dir: "/var/cache/boyan/covers"
  max_cache_mb: 256
```

### 4. Размещение статики фронтендов

Скопируйте содержимое собранных папок `dist` обоих фронтендов:

```bash
sudo cp -r web-desktop/* /var/www/boyan/web-desktop/
sudo cp -r web-mobile/* /var/www/boyan/web-mobile/
```

### 5. Настройка прав доступа

```bash
# Доступ для сервиса бэкенда
sudo chown -R boyan:boyan /opt/boyan /var/lib/boyan /var/cache/boyan
sudo chmod 750 /var/lib/boyan /var/cache/boyan

# Доступ для веб-сервера Nginx
sudo chown -R www-data:www-data /var/www/boyan
sudo chmod -R 755 /var/www/boyan
```

### 6. Настройка и запуск службы systemd

Создайте файл `/etc/systemd/system/boyan.service`:

```ini
[Unit]
Description=Next-Gen OPDS Suite (Boyan) Server
After=network.target

[Service]
Type=simple
User=boyan
Group=boyan
WorkingDirectory=/opt/boyan
ExecStart=/opt/boyan/boyan --config /opt/boyan/config.yaml
Restart=always
RestartSec=3s
LimitNOFILE=65535

# Песочница безопасности
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true
PrivateTmp=true

# Логирование в systemd journal
StandardOutput=journal
StandardError=journal
SyslogIdentifier=boyan

[Install]
WantedBy=multi-user.target
```

Активируйте и запустите службу:

```bash
sudo systemctl daemon-reload
sudo systemctl enable boyan
sudo systemctl start boyan

# Проверка статуса:
sudo systemctl status boyan
```

---

### 7. Настройка Nginx и HTTPS (SSL)

Создайте файл конфигурации виртуального хоста `/etc/nginx/sites-available/boyan`:

```nginx
upstream boyan_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    listen [::]:80;
    
    server_name books.example.com; # Укажите ваш домен

    client_max_body_size 200M;

    # Gzip сжатие
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml;

    # 1. Проксирование Backend API и OPDS каталогов
    location ~ ^/(api|opds|covers|health) {
        proxy_pass http://boyan_backend;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
        proxy_buffering off;
    }

    # 2. Мобильное PWA-приложение (/m/)
    location /m/ {
        alias /var/www/boyan/web-mobile/;
        try_files $uri $uri/ /m/index.html;

        location ~* (sw\.js|registerSW\.js|manifest\.webmanifest)$ {
            add_header Cache-Control "no-cache, no-store, must-revalidate";
            expires 0;
        }

        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }

    # 3. Десктопное веб-приложение (/)
    location / {
        root /var/www/boyan/web-desktop;
        index index.html;
        try_files $uri $uri/ /index.html;

        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }
}
```

Активируйте сайт и проверьте конфигурацию:

```bash
sudo ln -s /etc/nginx/sites-available/boyan /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

### 8. Бесплатный SSL-сертификат Let's Encrypt (HTTPS)

Для того чтобы PWA-приложение работало автономно и сохраняло книги в офлайн, **HTTPS строго обязателен**.

Выпустите сертификат с автоматической настройкой Nginx:

```bash
sudo certbot --nginx -d books.example.com
```

Certbot автоматически настроит перенаправление с HTTP на HTTPS и установит таймер автообновления сертификатов.

---

### 9. Настройка межсетевого экрана (UFW)

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

---

## Способ 3. Развертывание через Docker Compose

Если вы предпочитаете запуск сервисов в Docker-контейнерах:

1. Установите Docker и Docker Compose на VPS:

   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo apt install -y docker-compose-plugin
   ```

2. Клонируйте репозиторий:

   ```bash
   git clone https://github.com/ZAVHome/boyan.git /opt/boyan
   cd /opt/boyan
   ```

3. Скопируйте файл конфигурации:

   ```bash
   cp config.example.yaml config.yaml
   ```

4. Запустите стек:

   ```bash
   docker compose up -d --build
   ```

Контейнеры будут автоматически собирать образы на базе легковесного Alpine Linux и поднимать сетевые порты:

- `8080` — Core Go Backend
- `3000` — Web Desktop
- `3001` — Web Mobile PWA

---

## 🛠️ Эксплуатация и администрирование

### Добавление новых книг в библиотеку

- **Через папку импорта:**  
  Просто скопируйте книги (`.fb2`, `.fb2.zip`, `.epub`, `.mobi`, `.pdf`) в каталог `/var/lib/boyan/import/`:

  ```bash
  scp ~/Books/*.fb2.zip root@your-server-ip:/var/lib/boyan/import/
  ```

  Фоновый наблюдатель `fsnotify` мгновенно обнаружит новые файлы, проверит дубликаты и распакует метаданные в каталог.

- **Через веб-интерфейс:**  
  Авторизуйтесь администратором на сайте и нажмите кнопку **«Загрузить книгу»** в навигационной панели.

- **Через Telegram-бота:**  
  Отправьте файл книги документом прямо в диалог с ботом.

---

### Управление службой и просмотр логов

```bash
# Статус сервиса
sudo systemctl status boyan

# Перезапуск сервиса
sudo systemctl restart boyan

# Просмотр логов в реальном времени
sudo journalctl -u boyan -f

# Просмотр последних 100 строк лога
sudo journalctl -u boyan -n 100 --no-pager
```

---

### Резервное копирование базы данных (Backup)

База данных SQLite 3 работает в режиме `WAL` (Write-Ahead Logging). Горячий бэкап можно выполнять без остановки сервера с помощью встроенной команды SQLite:

```bash
# Создание мгновенной резервной копии БД
sqlite3 /var/lib/boyan/data/boyan.db ".backup '/var/lib/boyan/data/boyan_backup_$(date +%F).db'"
```

Добавьте задание в cron (`crontab -e`) для ежедневного бэкапа в 03:00 ночи:

```cron
0 3 * * * sqlite3 /var/lib/boyan/data/boyan.db ".backup '/var/lib/boyan/data/backup_\$(date +\%F).db'" && find /var/lib/boyan/data -name "backup_*.db" -mtime +14 -delete
```

---

### Обновление на новую версию «Бояна»

Благодаря внешней сборке обновление сервера занимает менее 10 секунд:

```bash
# 1. Распакуйте новый архив
tar -xzf /tmp/boyan-linux-amd64.tar.gz -C /tmp/boyan-new/

# 2. Обновите бинарник и статику
sudo cp /tmp/boyan-new/boyan /opt/boyan/boyan
sudo cp -r /tmp/boyan-new/web-desktop/* /var/www/boyan/web-desktop/
sudo cp -r /tmp/boyan-new/web-mobile/* /var/www/boyan/web-mobile/

# 3. Перезапустите службу
sudo systemctl restart boyan
```

Миграции схемы базы данных SQLite выполняются сервером автоматически при каждом запуске.
