@echo off
chcp 65001 >nul 2>&1
REM WinPHP 2025 - Launcher (auto-elevates to admin)

setlocal
set "ROOT=%~dp0"
set "PS1=%ROOT%WinPHP.ps1"

REM Check admin privileges
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -NoProfile -Command "Start-Process -Verb RunAs -FilePath '%~f0'"
    exit /b
)

REM Launch GUI (hidden console)
powershell -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File "%PS1%"
endlocal
