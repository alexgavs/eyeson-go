@echo off
echo ========================================
echo  Building and Starting EyesOn Server
echo ========================================
echo.

cd eyeson-go-server

echo [1/2] Building server...
go build -o eyeson-server.exe cmd/server/main.go

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ Build FAILED!
    echo Check the error messages above.
    echo.
    pause
    exit /b 1
)

echo.
echo ✅ Build successful!
echo.
echo [2/2] Starting server...
echo Server will start on http://localhost:5000
echo Press Ctrl+C to stop
echo.

eyeson-server.exe

pause
