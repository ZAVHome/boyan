# deploy-to-wsl.ps1: Сборка и автоматическое развертывание в WSL2 (Ubuntu)
$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path "$PSScriptRoot\.."

# 1. Сборка бинарника
& "$PSScriptRoot\build-wsl.ps1"

# 2. Преобразуем путь проекта в WSL путь (/mnt/d/...)
$Drive = $ProjectRoot.Drive.Name.ToLower()
$PathWithoutDrive = $ProjectRoot.Path.Substring(3).Replace('\', '/')
$WslPath = "/mnt/$Drive/$PathWithoutDrive"

Write-Host "===> Invoking WSL2 deployment script in Ubuntu ($WslPath)..." -ForegroundColor Cyan

# Запуск deploy.sh внутри WSL2
wsl -d Ubuntu -u root -- bash "$WslPath/scripts/wsl/deploy.sh"

Write-Host "===> Checking service status from Windows..." -ForegroundColor Green
try {
    $res = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get -TimeoutSec 5
    Write-Host "Server response:" -ForegroundColor Yellow
    $res | ConvertTo-Json
}
catch {
    Write-Host "Warning: Could not connect to http://localhost:8080/health yet: $_" -ForegroundColor DarkYellow
}
