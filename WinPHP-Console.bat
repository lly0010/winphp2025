@echo off
REM WinPHP - 调试模式 (保留控制台,可看到错误)
setlocal
set "ROOT=%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%ROOT%WinPHP.ps1"
pause
endlocal
