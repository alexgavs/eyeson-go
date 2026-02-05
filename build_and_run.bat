@echo off
chcp 65001 >nul
echo ========================================
echo  Building and Starting EyesOn Server
echo ========================================
echo.

cd /d "%~dp0"

REM Default environment (only if not already set)
if "%APP_ENV%"=="" set "APP_ENV=dev"
if "%EYESON_API_BASE_URL%"=="" set "EYESON_API_BASE_URL=https://eot-portal.pelephone.co.il:8888"
if "%EYESON_SIMULATOR_BASE_URL%"=="" set "EYESON_SIMULATOR_BASE_URL=http://127.0.0.1:8888"

REM Optional: rebuild UI before building the server
if "%BUILD_UI%"=="1" (
    echo [0/2] Rebuilding UI: BUILD_UI=1...
    set "NO_PAUSE=1"
    call rebuild_ui.bat
    if %ERRORLEVEL% NEQ 0 exit /b 1
)

cd eyeson-go-server

echo [1/2] Building server...
go build -o eyeson-server.exe ./cmd/server

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ Build FAILED!
    echo Check the error messages above.
    echo.
    if not "%NO_PAUSE%"=="1" pause
    exit /b 1
)

echo.
echo ✅ Build successful!
echo.
echo [2/2] Starting server...
echo Server will start on http://localhost:5000
echo Admin can switch upstream at /api/v1/upstream (UI applies after restart)
echo Press Ctrl+C to stop
echo.

if "%DETACHED%"=="1" (
    echo Starting server in background: DETACHED=1...
    start "EyesOn Server" /b eyeson-server.exe
    exit /b 0
)

eyeson-server.exe

if not "%NO_PAUSE%"=="1" pause
