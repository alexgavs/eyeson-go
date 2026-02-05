@echo off
chcp 65001 >nul
echo ========================================
echo  Rebuilding Frontend UI
echo ========================================
echo.

cd /d "%~dp0eyeson-gui\frontend"

echo [1/2] Installing dependencies (if needed)...
if not exist "node_modules" (
    call npm install
) else (
    echo node_modules exists - skipping npm install
)

echo.
echo [2/2] Building frontend...
call npm run build

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ❌ Build FAILED!
    if not "%NO_PAUSE%"=="1" pause
    exit /b 1
)

echo.
echo ✅ Frontend built successfully!
echo.
echo Copying to server static folder...
cd /d "%~dp0eyeson-go-server"
if exist "static\assets" (
    echo Cleaning old assets...
    rmdir /s /q "static\assets"
)
if not exist "static" mkdir "static"
if not exist "static\assets" mkdir "static\assets"

xcopy /E /I /Y /Q "%~dp0eyeson-gui\frontend\dist\assets" "static\assets" >nul
copy /Y "%~dp0eyeson-gui\frontend\dist\index.html" "static\index.html" >nul

echo.
echo ✅ UI updated! Refresh browser to see changes.
echo.

if not "%NO_PAUSE%"=="1" pause
