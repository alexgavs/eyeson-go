@echo off
chcp 65001 >nul
echo ========================================
echo  Starting Pelephone API Simulator
echo ========================================
echo.

cd /d "%~dp0"

if "%EYESON_SIMULATOR_BASE_URL%"=="" set "EYESON_SIMULATOR_BASE_URL=http://127.0.0.1:8888"

cd /d "%~dp0pelephone-simulator"

if not exist simulator.exe (
    echo Building simulator first...
    call build.bat
    if %ERRORLEVEL% NEQ 0 (
        echo ❌ Simulator build FAILED!
        if not "%NO_PAUSE%"=="1" pause
        exit /b 1
    )
)

echo 🎭 Pelephone API Simulator
echo 📊 Web Panel: http://localhost:8888/web
echo 🔌 API Port: 8888
echo 🌐 EYESON_SIMULATOR_BASE_URL=%EYESON_SIMULATOR_BASE_URL%
echo.
echo Press Ctrl+C to stop
echo.

if "%DETACHED%"=="1" (
    echo Starting simulator in background: DETACHED=1...
    start "Pelephone Simulator" /b simulator.exe
    exit /b 0
)

simulator.exe
