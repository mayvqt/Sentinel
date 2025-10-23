#!/usr/bin/env powershell

# Production startup script for Sentinel
# Copy this file and customize environment variables for your environment

param(
    [string]$Port = "8080",
    [string]$DatabaseURL = "sqlite://./sentinel.db",
    [string]$JWTSecret = ""
)

if ([string]::IsNullOrEmpty($JWTSecret)) {
    Write-Host "❌ Error: JWT_SECRET is required" -ForegroundColor Red
    Write-Host ""
    Write-Host "Usage:"
    Write-Host "  .\scripts\start-prod.ps1 -JWTSecret 'your-secure-secret-key'"
    Write-Host ""
    Write-Host "Or set environment variables:"
    Write-Host "  `$env:JWT_SECRET = 'your-secure-secret-key'"
    Write-Host "  .\scripts\start-prod.ps1"
    exit 1
}

# Set environment variables
$env:PORT = $Port
$env:DATABASE_URL = $DatabaseURL
$env:JWT_SECRET = $JWTSecret

Write-Host "🚀 Starting Sentinel Authentication Service (Production)" -ForegroundColor Green
Write-Host "Port: $env:PORT" -ForegroundColor Cyan
Write-Host "Database: $env:DATABASE_URL" -ForegroundColor Cyan
Write-Host ""

# Build if executable doesn't exist
if (-not (Test-Path "Sentinel.exe")) {
    Write-Host "📦 Building application..." -ForegroundColor Yellow
    go build .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Build failed" -ForegroundColor Red
        exit 1
    }
}

# Start the server
.\Sentinel.exe