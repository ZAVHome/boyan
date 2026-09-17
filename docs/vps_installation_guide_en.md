# Linux VPS Installation Guide for Boyan

This guide provides a comprehensive walkthrough for deploying **Boyan** on Linux Virtual Private Servers (VPS / VDS), specifically tailored for **Ubuntu 22.04 / 24.04 LTS**, **Debian 11 / 12**, and **Rocky Linux 9**.

---

## 💡 Deployment Philosophy: Zero-Build on Production VPS

Building web applications directly on production VPS servers (installing the Go compiler, Node.js, npm, downloading gigabytes of devDependencies) wastes scarce server RAM, creates security liabilities, and complicates system maintenance.

Boyan is engineered from the ground up for **external builds**:

1. **The Go Backend** compiles down to a single static binary without CGO (`CGO_ENABLED=0`) requiring zero external dependencies and consuming merely **~5 MB RAM**.
2. **The Frontends** (Desktop SPA and Mobile Touch PWA) compile to pure static HTML/CSS/JS bundles.
3. The server only receives a compact pre-built release package (~11–12 MB), immediately managed by `systemd` and served via `Nginx`.

---

## 🖥️ VPS System Requirements

| Specification | Minimum | Recommended |
| :--- | :--- | :--- |
| **CPU (vCPU)** | 1 core | 1–2 cores |
| **RAM** | 512 MB RAM | 1 GB RAM |
| **Disk Space** | 2 GB + book library | 10 GB + book library (SSD/NVMe) |
| **Operating System** | Ubuntu 22.04+, Debian 11+ | Ubuntu 24.04 LTS |
| **Network** | Public IPv4 / IPv6 | Dedicated subdomain (for HTTPS) |

---

## 🌐 DNS Preparation: Dedicated Subdomain (`books.MYDOMAIN.COM`)

For seamless desktop/mobile browser access, PWA offline capabilities (Service Workers strictly require HTTPS), and secure OPDS catalog sync with e-readers, Boyan is designed to run on a **dedicated subdomain** (e.g. `books.MYDOMAIN.COM`):

1. **Create an `A` Record in your DNS provider control panel:**
   - **Type:** `A`
   - **Host / Name:** `books` (or `books.MYDOMAIN.COM`)
   - **Value (Target):** Public IPv4 address of your VPS
   - **TTL:** `300` seconds (5 min) or provider default

2. **(Optional) `AAAA` Record for IPv6:**
   - Add an `AAAA` record if your VPS has a public IPv6 address.

3. **Verify DNS Propagation:**

   ```bash
   dig +short books.MYDOMAIN.COM
   # or
   nslookup books.MYDOMAIN.COM
   ```

---

## Method 1. One-Click Pre-Built Release Deployment (Recommended)

### Step 1. Package the Release on Developer Machine

On your local workstation (where Go and Node.js are already installed), run the packaging script:

- **Windows (PowerShell):**

  ```powershell
  powershell -ExecutionPolicy Bypass -File scripts/package-release.ps1
  ```

- **Linux / macOS / WSL2 (Bash):**

  ```bash
  bash scripts/package-release.sh
  ```

This packages the standalone release archive into `dist/boyan-linux-amd64.tar.gz` (approx. 11.6 MB).  
*(Alternatively, download the latest `boyan-linux-amd64.tar.gz` directly from [GitHub Releases](https://github.com/ZAVHome/boyan/releases)).*

---

### Step 2. Transfer the Archive to Your VPS

Upload the package via `scp`:

```bash
scp dist/boyan-linux-amd64.tar.gz root@YOUR_SERVER_IP:/tmp/
```

---

### Step 3. Execute Installation on VPS

Connect via SSH and run the installer with your dedicated subdomain:

```bash
ssh root@YOUR_SERVER_IP

# Create staging directory and extract archive
mkdir -p /tmp/boyan-pkg
tar -xzf /tmp/boyan-linux-amd64.tar.gz -C /tmp/boyan-pkg
cd /tmp/boyan-pkg

# Run automated installer with your subdomain
sudo bash install.sh books.MYDOMAIN.COM
```

**What `install.sh` handles automatically:**

1. Provisions an isolated unprivileged system user `boyan`.
2. Creates the directory hierarchy:
   - `/opt/boyan` — Server executable and `config.yaml`.
   - `/var/lib/boyan/data` — SQLite 3 WAL database.
   - `/var/lib/boyan/library` — Persistent book storage.
   - `/var/lib/boyan/import` — Auto-ingestion watch folder.
   - `/var/cache/boyan/covers` — In-memory LRU cover cache.
   - `/var/www/boyan/web-desktop` — Desktop web app static files.
   - `/var/www/boyan/web-mobile` — Mobile PWA reader static files.
3. Registers and starts the `systemd` service (`boyan.service`).
4. Configures the Nginx reverse proxy virtual host (`/etc/nginx/sites-available/boyan`) tailored for `books.MYDOMAIN.COM`.
5. Prompts to automatically obtain a free **Let's Encrypt SSL certificate** and enable HTTPS redirection.

---

## Method 2. Step-by-Step Manual Setup

If you prefer full control over every step or wish to use custom directory paths:

### 1. Update Packages & Install Dependencies

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y nginx curl certbot python3-certbot-nginx ufw
```

### 2. Create Unprivileged System User & Folders

```bash
sudo useradd -r -s /bin/false -d /opt/boyan boyan

sudo mkdir -p /opt/boyan
sudo mkdir -p /var/www/boyan/web-desktop
sudo mkdir -p /var/www/boyan/web-mobile
sudo mkdir -p /var/lib/boyan/data
sudo mkdir -p /var/lib/boyan/library
sudo mkdir -p /var/lib/boyan/import
sudo mkdir -p /var/cache/boyan/covers
```

### 3. Deploy Binary & Configuration

```bash
sudo cp boyan /opt/boyan/boyan
sudo chmod +x /opt/boyan/boyan

sudo cp config.example.yaml /opt/boyan/config.yaml
```

Update `/opt/boyan/config.yaml` with server paths:

```yaml
server:
  host: "127.0.0.1"    # Local interface only (proxied by Nginx)
  port: 8080
  jwt_secret: "SPECIFY_A_SECURE_RANDOM_32_CHAR_STRING"

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

### 4. Deploy Frontend Static Assets

```bash
sudo cp -r web-desktop/* /var/www/boyan/web-desktop/
sudo cp -r web-mobile/* /var/www/boyan/web-mobile/
```

### 5. Configure File Permissions

```bash
sudo chown -R boyan:boyan /opt/boyan /var/lib/boyan /var/cache/boyan
sudo chmod 750 /var/lib/boyan /var/cache/boyan

sudo chown -R www-data:www-data /var/www/boyan
sudo chmod -R 755 /var/www/boyan
```

### 6. Configure Systemd Service

Create `/etc/systemd/system/boyan.service`:

```ini
[Unit]
Description=Boyan Server
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

# Process sandbox
ProtectSystem=full
ProtectHome=true
NoNewPrivileges=true
PrivateTmp=true

# Logging to systemd journal
StandardOutput=journal
StandardError=journal
SyslogIdentifier=boyan

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable boyan
sudo systemctl start boyan
sudo systemctl status boyan
```

---

### 7. Configure Nginx Reverse Proxy

Create the virtual host configuration `/etc/nginx/sites-available/boyan` for your subdomain (e.g. `books.MYDOMAIN.COM`):

```nginx
upstream boyan_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    listen [::]:80;
    
    # Dedicated subdomain for Boyan
    server_name books.MYDOMAIN.COM;

    client_max_body_size 200M;

    # Let's Encrypt ACME challenge validation
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml;

    # 1. Reverse Proxy API & OPDS endpoints
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

    # 2. Mobile Touch PWA reader (/m/)
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

    # 3. Desktop Web App (/)
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

Enable site and reload Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/boyan /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

### 8. Enable Let's Encrypt HTTPS (SSL)

HTTPS is strictly mandatory for Service Workers, IndexedDB offline book storage in PWA, and secure remote OPDS feed syncing.

Request a certificate and automatically configure Nginx:

```bash
sudo certbot --nginx -d books.MYDOMAIN.COM
```

**What Certbot does:**

1. Validates domain control for `books.MYDOMAIN.COM` with Let's Encrypt via HTTP-01 ACME challenge.
2. Generates trusted TLS certificates at `/etc/letsencrypt/live/books.MYDOMAIN.COM/`.
3. Modifies `/etc/nginx/sites-available/boyan` to configure port 443 SSL directives and creates an automatic 301 HTTP-to-HTTPS redirect.
4. Registers `certbot.timer` systemd timer for automatic renewals every 60 days.

Verify auto-renewal:

```bash
sudo certbot renew --dry-run
```

---

### 9. Firewall Configuration (UFW)

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

---

## Method 3. Docker Compose Deployment

If your infrastructure relies on Docker containers:

1. Install Docker and Docker Compose on your VPS:

   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo apt install -y docker-compose-plugin
   ```

2. Clone repository:

   ```bash
   git clone https://github.com/ZAVHome/boyan.git /opt/boyan
   cd /opt/boyan
   ```

3. Prepare configuration:

   ```bash
   cp config.example.yaml config.yaml
   ```

4. Start the stack:

   ```bash
   docker compose up -d --build
   ```

Containers will compile on lightweight Alpine Linux and expose:

- `8080` — Core Go Backend
- `3000` — Web Desktop
- `3001` — Web Mobile PWA

---

## 🛠️ Operations & Administration

### Ingesting New Books

- **Drop into Import folder:**  
  Copy books (`.fb2`, `.fb2.zip`, `.epub`, `.mobi`, `.pdf`) directly into `/var/lib/boyan/import/`:

  ```bash
  scp ~/Books/*.fb2.zip root@YOUR_SERVER_IP:/var/lib/boyan/import/
  ```

  The `fsnotify` watcher automatically processes, extracts metadata, caches covers, and registers records.

- **Web Upload:**  
  Log in as admin and click **"Upload Book"** in the top navigation bar.

- **Telegram Bot:**  
  Forward or send book files directly to your bot.

---

### Service Control & Logs

```bash
# Check service status
sudo systemctl status boyan

# Restart service
sudo systemctl restart boyan

# Tail real-time structured logs
sudo journalctl -u boyan -f

# View last 100 log lines
sudo journalctl -u boyan -n 100 --no-pager
```

---

### Hot Backups (SQLite WAL)

Run non-blocking online backups using SQLite's native backup utility:

```bash
sqlite3 /var/lib/boyan/data/boyan.db ".backup '/var/lib/boyan/data/boyan_backup_$(date +%F).db'"
```

Add a daily cron job (`crontab -e`):

```cron
0 3 * * * sqlite3 /var/lib/boyan/data/boyan.db ".backup '/var/lib/boyan/data/backup_\$(date +\%F).db'" && find /var/lib/boyan/data -name "backup_*.db" -mtime +14 -delete
```

---

### Updating Boyan

Because of the zero-build release package, updating takes under 10 seconds:

```bash
tar -xzf /tmp/boyan-linux-amd64.tar.gz -C /tmp/boyan-new/

sudo cp /tmp/boyan-new/boyan /opt/boyan/boyan
sudo cp -r /tmp/boyan-new/web-desktop/* /var/www/boyan/web-desktop/
sudo cp -r /tmp/boyan-new/web-mobile/* /var/www/boyan/web-mobile/

sudo systemctl restart boyan
```

Database schema migrations run automatically on server start.
