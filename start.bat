@echo off
chcp 65001 >nul
title EyesOn Server

echo ═══════════════════════════════════════════════════════════
echo              EyesOn SIM Management Server
echo                Full Build & Start Script
echo ═══════════════════════════════════════════════════════════
echo.

:: 1. Check Prerequisites
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH
    pause
    exit /b 1
)
where npm >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js/npm is not installed or not in PATH
    pause
    exit /b 1
)

:: 2. Build Frontend
echo [1/3] Building Frontend/UI...
cd /d "%~dp0eyeson-gui\frontend"
if not exist "node_modules" (
    echo       Installing dependencies...
    call npm install
)
echo       Running Vite build...
call npm run build
if %errorlevel% neq 0 (
    echo [ERROR] Frontend build failed!
    pause
    exit /b 1
)

:: 3. Prepare Static Assets
echo [2/3] Updating Server Static Files...
cd /d "%~dp0eyeson-go-server"
if exist "static\assets" (
    echo       Cleaning old assets...
    rmdir /s /q "static\assets"
)
if not exist "static" mkdir "static"
if not exist "static\assets" mkdir "static\assets"

echo       Copying new build files...
xcopy /E /I /Y /Q "%~dp0eyeson-gui\frontend\dist\assets" "static\assets"
copy /Y "%~dp0eyeson-gui\frontend\dist\index.html" "static\index.html" >nul

:: 4. Build & Start Server
echo [3/3] Building & Starting Go Server...
go mod tidy
go build -o server.exe ./cmd/server
if %errorlevel% neq 0 (
    echo [ERROR] Server build failed!
    pause
    exit /b 1
)

echo.
echo ───────────────────────────────────────────────────────────
echo   Web UI:     http://localhost:5000
echo   API Docs:   http://localhost:5000/docs
echo ───────────────────────────────────────────────────────────
echo.
echo Press Ctrl+C to stop the server
echo.

server.exe