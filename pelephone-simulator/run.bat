@echo off
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
echo.
echo Press Ctrl+C to stop
echo.

simulator.exe
