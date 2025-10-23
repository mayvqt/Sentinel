#!/usr/bin/env powershell

# Set environment variables
$env:PORT = "8080"
$env:DATABASE_URL = "sqlite://./sentinel.db"
$env:JWT_SECRET = "YkBm/hZZYyq8VHQ4caGT8m22VJH/F02fPDKvCFuNTuo="

Write-Host "🚀 Starting Sentinel Authentication Service"
Write-Host "Port: $env:PORT"
Write-Host "Database: $env:DATABASE_URL"
Write-Host ""

# Start the server using main.go
go run .