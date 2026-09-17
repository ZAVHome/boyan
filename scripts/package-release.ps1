# ==============================================================================
# Script to build autonomous release bundle for Linux VPS (Windows PowerShell)
# Output: dist/boyan-linux-amd64.tar.gz
# ==============================================================================
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$DistDir = Join-Path $RootDir "dist"
$StagingDir = Join-Path $DistDir "boyan-linux-amd64"
$ArchiveFile = Join-Path $DistDir "boyan-linux-amd64.tar.gz"

Write-Host "==> Cleaning previous builds..." -ForegroundColor Cyan
if (Test-Path $StagingDir) { Remove-Item -Recurse -Force $StagingDir }
if (Test-Path $ArchiveFile) { Remove-Item -Force $ArchiveFile }
New-Item -ItemType Directory -Force -Path $StagingDir | Out-Null

Write-Host "==> 1. Building Go backend for Linux (amd64, CGO_ENABLED=0)..." -ForegroundColor Cyan
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
Push-Location (Join-Path $RootDir "backend")
try {
    go build -ldflags="-s -w" -o (Join-Path $StagingDir "boyan") ./cmd/server
} finally {
    Pop-Location
}

Write-Host "==> 2. Building Desktop Web Frontend (web-desktop)..." -ForegroundColor Cyan
Push-Location (Join-Path $RootDir "frontends/web-desktop")
try {
    npm run build
    Copy-Item -Recurse -Force "dist" (Join-Path $StagingDir "web-desktop")
} finally {
    Pop-Location
}

Write-Host "==> 3. Building Mobile PWA Frontend (web-mobile)..." -ForegroundColor Cyan
Push-Location (Join-Path $RootDir "frontends/web-mobile")
try {
    npm run build
    Copy-Item -Recurse -Force "dist" (Join-Path $StagingDir "web-mobile")
} finally {
    Pop-Location
}

Write-Host "==> 4. Copying configuration templates and installer..." -ForegroundColor Cyan
Copy-Item (Join-Path $RootDir "config.example.yaml") (Join-Path $StagingDir "config.example.yaml")
Copy-Item (Join-Path $RootDir "scripts/systemd/boyan.service") (Join-Path $StagingDir "boyan.service")
Copy-Item (Join-Path $RootDir "scripts/nginx/boyan.conf") (Join-Path $StagingDir "nginx-boyan.conf")
Copy-Item (Join-Path $RootDir "scripts/installer/install.sh") (Join-Path $StagingDir "install.sh")

Write-Host "==> 5. Creating tar.gz archive..." -ForegroundColor Cyan
Push-Location $DistDir
try {
    tar -czf "boyan-linux-amd64.tar.gz" -C $StagingDir .
} finally {
    Pop-Location
}

$Size = [math]::Round(((Get-Item $ArchiveFile).Length / 1MB), 2)
Write-Host ""
Write-Host "SUCCESS: Release package for Linux VPS created!" -ForegroundColor Green
Write-Host "Archive: $ArchiveFile ($Size MB)" -ForegroundColor Green
Write-Host "To deploy to your VPS:" -ForegroundColor Yellow
Write-Host "  scp dist/boyan-linux-amd64.tar.gz root@your-vps:/tmp/" -ForegroundColor Yellow
Write-Host "  ssh root@your-vps" -ForegroundColor Yellow
Write-Host "  mkdir -p /tmp/boyan-pkg && tar -xzf /tmp/boyan-linux-amd64.tar.gz -C /tmp/boyan-pkg" -ForegroundColor Yellow
Write-Host "  cd /tmp/boyan-pkg && sudo bash install.sh books.MYDOMAIN.COM" -ForegroundColor Yellow
