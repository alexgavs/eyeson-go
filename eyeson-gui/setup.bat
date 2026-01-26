@echo off
echo ==========================================
echo Setting up EyesOn GUI Project
echo ==========================================

echo.
echo [1/3] Checking requirements...
go version
if %errorlevel% neq 0 (
    echo Error: Go is not installed or not in PATH.
    pause
    exit /b 1
)
node -v
if %errorlevel% neq 0 (
    echo Error: Node.js is not installed or not in PATH.
    pause
    exit /b 1
)

echo.
echo [2/3] Installing Go dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo Error installing Go dependencies.
    pause
    exit /b 1
)

echo.
echo [3/3] Installing Frontend dependencies...
cd frontend
call npm install
if %errorlevel% neq 0 (
    echo Error installing NPM packages.
    cd ..
    pause
    exit /b 1
)
cd ..

echo.
echo ==========================================
echo Setup Complete!
echo ==========================================
echo.
echo To run the application in development mode:
echo   wails dev
echo.
echo To build the production binary:
echo   wails build
echo.
pause
