@echo off
REM Quick run script - assumes build.bat has been run at least once
REM This script loads environment variables from .env file and starts the server

if not exist moon.exe (
    echo moon.exe not found. Running build.bat first...
    call build.bat
    exit /b
)

REM Check if .env file exists
if not exist .env (
    echo ERROR: .env file not found!
    echo Please run build.bat first to create it.
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
    pause
    exit /b 1
)

echo Starting server on http://localhost:8484
echo Press Ctrl+C to stop the server
echo.

moon.exe
