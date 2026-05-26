@echo off
chcp 65001 >nul 2>&1
REM WinPHP - Debug mode (keep console open to see errors)
setlocal
set "ROOT=%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%ROOT%WinPHP.ps1"
pause
endlocal
