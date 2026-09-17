# build-wsl.ps1: Кросс-компиляция бэкенда Boyan под Linux (WSL2)
$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path "$PSScriptRoot\.."
$BackendDir = "$ProjectRoot\backend"
$BinDir = "$ProjectRoot\bin"

Write-Host "===> Building Boyan Linux binary (GOOS=linux GOARCH=amd64 CGO_ENABLED=0)..." -ForegroundColor Cyan

if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
}

Push-Location $BackendDir
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"

    go build -ldflags="-s -w" -o "$BinDir\boyan-linux" ./cmd/server
    Write-Host "===> Build completed successfully: $BinDir\boyan-linux" -ForegroundColor Green
}
finally {
    Pop-Location
    $env:GOOS = ""
    $env:GOARCH = ""
    $env:CGO_ENABLED = ""
}
