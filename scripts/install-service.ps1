# install-service.ps1 - 可选: 将 Nginx / MySQL 注册为 Windows 服务,实现开机自启动
# 必须以管理员身份运行. 用法: powershell -ExecutionPolicy Bypass -File install-service.ps1 [-Action install|uninstall]

param(
    [ValidateSet('install', 'uninstall', 'status')]
    [string]$Action = 'install'
)

$ErrorActionPreference = 'Stop'
$root = (Get-Item $PSScriptRoot).Parent.FullName
. (Join-Path $root 'src\Common.ps1')

$svcNginx = 'WinPHPNginx'
$svcMysql = 'WinPHPMySQL'

function Test-Admin {
    ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Admin)) { Write-Host '请以管理员身份运行' -ForegroundColor Red; exit 1 }

switch ($Action) {
    'install' {
        # Nginx 需要 srvany 或 nssm 才能正确作为服务运行, 这里使用内置 sc + 第三方包装方法
        Write-Host "提示: Windows 服务化建议使用 NSSM (https://nssm.cc/)" -ForegroundColor Yellow
        Write-Host "请下载 nssm.exe 放到面板的 bin/ 目录后再执行" -ForegroundColor Yellow
        $nssm = Join-Path $WP_BinDir 'nssm.exe'
        if (-not (Test-Path $nssm)) {
            Write-Host "未找到 $nssm,跳过" -ForegroundColor Red
            exit 1
        }

        # MySQL
        $mysqld = Join-Path $WP_MysqlDir 'bin\mysqld.exe'
        $myini  = Join-Path $WP_MysqlDir 'my.ini'
        if (Test-Path $mysqld) {
            & $nssm install $svcMysql $mysqld "--defaults-file=$myini"
            & $nssm set $svcMysql Start SERVICE_AUTO_START
            Write-Host "已安装服务: $svcMysql" -ForegroundColor Green
        }

        # Nginx
        $nginx = Join-Path $WP_NginxDir 'nginx.exe'
        if (Test-Path $nginx) {
            & $nssm install $svcNginx $nginx "-p" "$WP_NginxDir"
            & $nssm set $svcNginx Start SERVICE_AUTO_START
            & $nssm set $svcNginx AppDirectory $WP_NginxDir
            Write-Host "已安装服务: $svcNginx" -ForegroundColor Green
        }

        Write-Host '完成. 可通过 services.msc 管理.'
    }
    'uninstall' {
        $nssm = Join-Path $WP_BinDir 'nssm.exe'
        if (Test-Path $nssm) {
            & $nssm stop    $svcNginx 2>$null
            & $nssm remove  $svcNginx confirm 2>$null
            & $nssm stop    $svcMysql 2>$null
            & $nssm remove  $svcMysql confirm 2>$null
        } else {
            sc.exe stop $svcNginx 2>$null
            sc.exe delete $svcNginx 2>$null
            sc.exe stop $svcMysql 2>$null
            sc.exe delete $svcMysql 2>$null
        }
        Write-Host '已卸载服务' -ForegroundColor Green
    }
    'status' {
        foreach ($n in @($svcNginx, $svcMysql)) {
            $s = Get-Service -Name $n -ErrorAction SilentlyContinue
            if ($s) { Write-Host "$n : $($s.Status)" } else { Write-Host "$n : 未安装" }
        }
    }
}
