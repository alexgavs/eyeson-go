@echo off
chcp 65001 >nul
echo ========================================
echo  Pelephone API Simulator
echo ========================================
echo.

cd /d "%~dp0"

if not exist simulator.exe (
    echo Building simulator first...
    call build.bat
)

echo Starting simulator on port 8888...
echo.
echo API Endpoint: http://localhost:8888
echo Admin Panel:  http://localhost:8888/web
if "%EYESON_SIMULATOR_BASE_URL%"=="" set "EYESON_SIMULATOR_BASE_URL=http://127.0.0.1:8888"
echo EYESON_SIMULATOR_BASE_URL=%EYESON_SIMULATOR_BASE_URL%
echo.
echo Press Ctrl+C to stop
echo.

simulator.exe
