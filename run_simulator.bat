@echo off
echo ========================================
echo  Starting Pelephone API Simulator
echo ========================================
echo.

cd pelephone-simulator

if not exist pelephone-simulator.exe (
    echo ❌ Simulator not built yet!
    echo Please run build_simulator.bat first
    pause
    exit /b 1
)

echo 🎭 Pelephone API Simulator
echo 📊 Web Panel: http://localhost:8888/web
echo 🔌 API Port: 8888
echo.
echo Press Ctrl+C to stop
echo.

pelephone-simulator.exe
