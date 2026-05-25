@echo off
REM WinPHP 2025 - 启动入口
REM 自动提升管理员权限,以便修改 hosts、绑定 80 端口等

setlocal
set "ROOT=%~dp0"
set "PS1=%ROOT%WinPHP.ps1"

REM 检测管理员权限
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo 正在请求管理员权限...
    powershell -NoProfile -Command "Start-Process -Verb RunAs -FilePath '%~f0'"
    exit /b
)

REM 启动 GUI (隐藏控制台)
powershell -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File "%PS1%"
endlocal
