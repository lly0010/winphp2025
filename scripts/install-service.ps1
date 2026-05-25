# install-service.ps1 - CLI 入口: 一键启用/禁用全部开机自启
# 与面板"自启动"标签页的"一键启用全部"等价, 适合无界面环境或批处理使用.
# 用法:
#   powershell -ExecutionPolicy Bypass -File install-service.ps1 -Action enable
#   powershell -ExecutionPolicy Bypass -File install-service.ps1 -Action disable
#   powershell -ExecutionPolicy Bypass -File install-service.ps1 -Action status

param(
    [ValidateSet('enable', 'disable', 'status')]
    [string]$Action = 'status'
)

$ErrorActionPreference = 'Stop'
$root = (Get-Item $PSScriptRoot).Parent.FullName
$srcDir = Join-Path $root 'src'
. (Join-Path $srcDir 'Common.ps1')
. (Join-Path $srcDir 'Downloader.ps1')
. (Join-Path $srcDir 'Services.ps1')
. (Join-Path $srcDir 'Sites.ps1')
. (Join-Path $srcDir 'AutoStart.ps1')

Initialize-WPDirs

function Test-Admin {
    ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
if (-not (Test-Admin)) {
    Write-Host '请以管理员身份运行' -ForegroundColor Red; exit 1
}

switch ($Action) {
    'enable' {
        Write-Host '正在启用全部开机自启 (首次自动下载 NSSM)...' -ForegroundColor Cyan
        $r = Enable-WPAllAutoStart
        Write-Host ''
        Write-Host "面板自启: $(if ($r.Panel) {'✓'} else {'✗'})"
        foreach ($t in 'Nginx','Php','Mysql') {
            if ($r[$t] -eq $null) { Write-Host "$t : (组件未安装, 跳过)" -ForegroundColor Yellow }
            elseif ($r[$t])       { Write-Host "$t : ✓ 已注册" -ForegroundColor Green }
            else                  { Write-Host "$t : ✗ 失败" -ForegroundColor Red }
        }
    }
    'disable' {
        Write-Host '正在禁用全部开机自启...' -ForegroundColor Cyan
        Disable-WPAllAutoStart | Out-Null
        Write-Host '✓ 已禁用' -ForegroundColor Green
    }
    'status' {
        Write-Host "面板自启 (任务 $WP_TaskName): " -NoNewline
        if (Get-WPPanelAutoStartStatus) {
            Write-Host '✓ 启用' -ForegroundColor Green
        } else {
            Write-Host '✗ 未启用' -ForegroundColor Yellow
        }
        foreach ($n in @($WP_SvcNginx, $WP_SvcPhp, $WP_SvcMysql)) {
            $info = Get-WPServiceInfo $n
            Write-Host "$n : " -NoNewline
            if ($info.Installed) {
                Write-Host "$($info.Status) ($($info.StartType))" -ForegroundColor Green
            } else {
                Write-Host '未注册' -ForegroundColor Yellow
            }
        }
    }
}
