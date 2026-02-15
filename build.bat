@echo off
REM Build and run Moon application
REM This script loads environment variables from .env file and starts the server

echo Building Moon application...
go build -o moon.exe
if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful!
echo.

REM Check if .env file exists
if not exist .env (
    echo WARNING: .env file not found!
    echo Please copy .env.example to .env and add your Google Maps API key.
    echo.
    echo Creating .env from .env.example...
    copy .env.example .env
    echo.
    echo Please edit .env and add your GOOGLE_MAPS_API_KEY, then run this script again.
    pause
    exit /b 1
)

echo Loading environment variables from .env...
for /f "usebackq tokens=1,* delims==" %%a in (.env) do (
    REM Skip comments and empty lines
    echo %%a | findstr /r "^#" >nul
    if errorlevel 1 (
        if not "%%a"=="" (
            set "%%a=%%b"
        )
    )
)

REM Check if API key is set
if "%GOOGLE_MAPS_API_KEY%"=="" (
    echo ERROR: GOOGLE_MAPS_API_KEY is not set in .env file!
    echo Please edit .env and add your Google Maps API key.
    pause
    exit /b 1
)

if "%GOOGLE_MAPS_API_KEY%"=="your_google_maps_api_key_here" (
    echo ERROR: Please replace the placeholder API key in .env with your actual key!
    pause
    exit /b 1
)

echo API Key loaded: %GOOGLE_MAPS_API_KEY:~0,10%...
echo Production mode: %PROD%
echo.
echo Starting server on http://localhost:8181
echo Press Ctrl+C to stop the server
echo.

moon.exe
