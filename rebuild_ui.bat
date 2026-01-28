@echo off
echo ========================================
echo  Rebuilding Frontend UI
echo ========================================
echo.

cd eyeson-gui\frontend

echo [1/2] Installing dependencies (if needed)...
call npm install

echo.
echo [2/2] Building frontend...
call npm run build

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ Build FAILED!
    pause
    exit /b 1
)

echo.
echo ✅ Frontend built successfully!
echo.
echo Copying to server static folder...
xcopy /E /Y dist\* ..\..\eyeson-go-server\static\

echo.
echo ✅ UI updated! Refresh browser to see changes.
echo.

pause
