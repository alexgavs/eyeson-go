@echo off
echo ========================================
echo  Building Pelephone API Simulator
echo ========================================
echo.

cd /d "%~dp0"

echo [1/2] Downloading dependencies...
go mod tidy

echo.
echo [2/2] Building simulator...
go build -o simulator.exe main.go

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [X] Build FAILED!
    if not "%NO_PAUSE%"=="1" pause
    exit /b 1
)

echo.
echo [OK] Build successful!
echo.
echo Run: run_simulator.bat
if not "%NO_PAUSE%"=="1" pause
