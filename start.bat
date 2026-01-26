@echo off
chcp 65001 >nul
title EyesOn Server

echo ═══════════════════════════════════════════════════════════
echo              EyesOn SIM Management Server
echo ═══════════════════════════════════════════════════════════
echo.

cd /d "%~dp0eyeson-go-server"

:: Check if Go is installed
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH
    pause
    exit /b 1
)

:: Build the server
echo [1/2] Building server...
go build -o server.exe ./cmd/server
if %errorlevel% neq 0 (
    echo [ERROR] Build failed!
    pause
    exit /b 1
)
echo [OK] Build successful

:: Start the server
echo [2/2] Starting server on port 5000...
echo.
echo ───────────────────────────────────────────────────────────
echo   Web UI:     http://localhost:5000
echo   API Docs:   http://localhost:5000/docs
echo ───────────────────────────────────────────────────────────
echo.
echo Press Ctrl+C to stop the server
echo.

server.exe
