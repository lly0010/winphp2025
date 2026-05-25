# Services.ps1 - Nginx / PHP-CGI / MySQL 服务控制
# 全部以前台进程方式启动并隐藏窗口，由面板管理 PID

# ---------- 配置初始化 ----------
function Initialize-WPNginxConfig {
    $nginxConfDir = Join-Path $WP_NginxDir 'conf'
    if (-not (Test-Path $nginxConfDir)) { return }

    $tpl = Join-Path $WP_TplDir 'nginx.conf'
    $text = Expand-WPTemplate -TemplatePath $tpl -Tokens @{
        WWW_ROOT = $WP_WwwDir.Replace('\','/')
    }
    Set-Content -Path (Join-Path $nginxConfDir 'nginx.conf') -Value $text -Encoding ASCII

    $vhostDir = Join-Path $nginxConfDir 'vhosts'
    if (-not (Test-Path $vhostDir)) { New-Item -ItemType Directory $vhostDir -Force | Out-Null }

    $logDir = Join-Path $WP_NginxDir 'logs'
    if (-not (Test-Path $logDir)) { New-Item -ItemType Directory $logDir -Force | Out-Null }

    # 写一个默认首页
    $idx = Join-Path $WP_WwwDir 'default\index.php'
    if (-not (Test-Path $idx)) {
        @'
<?php
echo "<h1>WinPHP - It works!</h1>";
echo "<p>PHP Version: " . phpversion() . "</p>";
echo "<p>Server: " . ($_SERVER['SERVER_SOFTWARE'] ?? '') . "</p>";
echo "<p>Document Root: " . $_SERVER['DOCUMENT_ROOT'] . "</p>";
echo "<hr><a href='phpinfo.php'>phpinfo()</a>";
'@ | Out-File $idx -Encoding UTF8
    }
    $info = Join-Path $WP_WwwDir 'default\phpinfo.php'
    if (-not (Test-Path $info)) { '<?php phpinfo();' | Out-File $info -Encoding UTF8 }

    Write-WPLog 'Nginx 配置已初始化'
}

function Initialize-WPPhpConfig {
    if (-not (Test-Path $WP_PhpDir)) { return }

    $iniSample = Join-Path $WP_PhpDir 'php.ini-production'
    $iniTarget = Join-Path $WP_PhpDir 'php.ini'

    $tpl = Join-Path $WP_TplDir 'php.ini'
    $text = Expand-WPTemplate -TemplatePath $tpl -Tokens @{
        PHP_DIR = $WP_PhpDir.Replace('\','/')
    }
    Set-Content -Path $iniTarget -Value $text -Encoding ASCII

    foreach ($d in @('logs','tmp')) {
        $p = Join-Path $WP_PhpDir $d
        if (-not (Test-Path $p)) { New-Item -ItemType Directory $p -Force | Out-Null }
    }
    Write-WPLog 'PHP 配置已初始化'
}

function Initialize-WPMysqlConfig {
    if (-not (Test-Path $WP_MysqlDir)) { return }

    $iniTarget = Join-Path $WP_MysqlDir 'my.ini'
    $tpl = Join-Path $WP_TplDir 'my.ini'
    $text = Expand-WPTemplate -TemplatePath $tpl -Tokens @{
        MYSQL_DIR = $WP_MysqlDir.Replace('\','/')
    }
    Set-Content -Path $iniTarget -Value $text -Encoding ASCII

    foreach ($d in @('logs','tmp')) {
        $p = Join-Path $WP_MysqlDir $d
        if (-not (Test-Path $p)) { New-Item -ItemType Directory $p -Force | Out-Null }
    }
    Write-WPLog 'MySQL 配置已初始化'
}

# ---------- Nginx ----------
function Get-WPNginxStatus {
    $procs = Get-WPProcess -Name 'nginx' -PathFilter $WP_NginxDir
    return [pscustomobject]@{
        Running = ($procs.Count -gt 0)
        Procs   = $procs
        Version = Get-NginxRunningVersion
    }
}

function Start-WPNginx {
    $exe = Join-Path $WP_NginxDir 'nginx.exe'
    if (-not (Test-Path $exe)) { Write-WPLog 'Nginx 未安装' 'WARN'; return $false }

    $status = Get-WPNginxStatus
    if ($status.Running) { Write-WPLog 'Nginx 已在运行' 'WARN'; return $true }

    # 配置语法检查
    $test = & $exe -t -p $WP_NginxDir 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-WPLog "Nginx 配置检查失败: $test" 'ERROR'
        return $false
    }
    Start-Process -FilePath $exe -ArgumentList "-p `"$WP_NginxDir`"" -WorkingDirectory $WP_NginxDir -WindowStyle Hidden | Out-Null
    Start-Sleep -Milliseconds 600
    $ok = (Get-WPNginxStatus).Running
    Write-WPLog ("Nginx 启动 " + $(if ($ok) {'成功'} else {'失败'}))
    return $ok
}

function Stop-WPNginx {
    $exe = Join-Path $WP_NginxDir 'nginx.exe'
    if (Test-Path $exe) {
        & $exe -s stop -p $WP_NginxDir 2>&1 | Out-Null
    }
    Start-Sleep -Milliseconds 400
    Get-WPProcess -Name 'nginx' -PathFilter $WP_NginxDir | ForEach-Object {
        try { Stop-Process -Id $_.Id -Force } catch {}
    }
    Write-WPLog 'Nginx 已停止'
    return $true
}

function Restart-WPNginx {
    Stop-WPNginx | Out-Null
    Start-Sleep -Milliseconds 300
    return Start-WPNginx
}

function Invoke-WPNginxReload {
    $exe = Join-Path $WP_NginxDir 'nginx.exe'
    if (-not (Test-Path $exe)) { return $false }
    $test = & $exe -t -p $WP_NginxDir 2>&1
    if ($LASTEXITCODE -ne 0) { Write-WPLog "配置错误: $test" 'ERROR'; return $false }
    & $exe -s reload -p $WP_NginxDir 2>&1 | Out-Null
    Write-WPLog 'Nginx 已重载配置'
    return $true
}

# ---------- PHP-CGI ----------
function Get-WPPhpStatus {
    $procs = Get-WPProcess -Name 'php-cgi' -PathFilter $WP_PhpDir
    return [pscustomobject]@{
        Running = ($procs.Count -gt 0)
        Procs   = $procs
        Version = Get-PHPInstalledVersion
    }
}

function Start-WPPhp {
    $exe = Join-Path $WP_PhpDir 'php-cgi.exe'
    if (-not (Test-Path $exe)) { Write-WPLog 'PHP 未安装' 'WARN'; return $false }

    $status = Get-WPPhpStatus
    if ($status.Running) { Write-WPLog 'PHP-CGI 已在运行' 'WARN'; return $true }

    if (Test-WPPort 9000) {
        Write-WPLog '端口 9000 被占用，启动失败' 'ERROR'; return $false
    }

    $ini = Join-Path $WP_PhpDir 'php.ini'
    # 启动多个 PHP-CGI 子进程，使用 PHP_FCGI_CHILDREN/MAX_REQUESTS
    $env:PHP_FCGI_MAX_REQUESTS = '1000'
    $env:PHP_FCGI_CHILDREN     = '5'
    Start-Process -FilePath $exe `
        -ArgumentList "-b","127.0.0.1:9000","-c","`"$ini`"" `
        -WorkingDirectory $WP_PhpDir -WindowStyle Hidden | Out-Null
    Start-Sleep -Milliseconds 600
    $ok = (Get-WPPhpStatus).Running
    Write-WPLog ("PHP-CGI 启动 " + $(if ($ok) {'成功'} else {'失败'}))
    return $ok
}

function Stop-WPPhp {
    Get-WPProcess -Name 'php-cgi' -PathFilter $WP_PhpDir | ForEach-Object {
        try { Stop-Process -Id $_.Id -Force } catch {}
    }
    Write-WPLog 'PHP-CGI 已停止'
    return $true
}

function Restart-WPPhp {
    Stop-WPPhp | Out-Null
    Start-Sleep -Milliseconds 300
    return Start-WPPhp
}

# ---------- MySQL ----------
function Get-WPMysqlStatus {
    $procs = Get-WPProcess -Name 'mysqld' -PathFilter $WP_MysqlDir
    return [pscustomobject]@{
        Running = ($procs.Count -gt 0)
        Procs   = $procs
        Version = Get-MySQLInstalledVersion
    }
}

function Initialize-WPMysqlData {
    # 首次启动前必须初始化 data 目录
    $exe = Join-Path $WP_MysqlDir 'bin\mysqld.exe'
    $ini = Join-Path $WP_MysqlDir 'my.ini'
    $dataDir = Join-Path $WP_MysqlDir 'data'

    if (Test-Path (Join-Path $dataDir 'mysql')) {
        Write-WPLog 'MySQL data 目录已存在，跳过初始化'
        return $true
    }
    if (Test-Path $dataDir) { Remove-Item $dataDir -Recurse -Force }

    Write-WPLog 'MySQL 正在初始化 data 目录（首次需 1~2 分钟）...'
    $mysqlArgs = "--defaults-file=`"$ini`" --initialize-insecure --console"
    $p = Start-Process -FilePath $exe -ArgumentList $mysqlArgs -WorkingDirectory $WP_MysqlDir -Wait -PassThru -WindowStyle Hidden
    if ($p.ExitCode -ne 0) {
        Write-WPLog "MySQL 初始化失败，退出码 $($p.ExitCode)" 'ERROR'
        return $false
    }
    $state = Get-WPState
    $state.mysqlInited = $true
    Save-WPState $state
    Write-WPLog 'MySQL 初始化完成，root 密码为空'
    return $true
}

function Start-WPMysql {
    $exe = Join-Path $WP_MysqlDir 'bin\mysqld.exe'
    if (-not (Test-Path $exe)) { Write-WPLog 'MySQL 未安装' 'WARN'; return $false }

    if ((Get-WPMysqlStatus).Running) { Write-WPLog 'MySQL 已在运行' 'WARN'; return $true }

    $state = Get-WPState
    if (-not $state.mysqlInited -or -not (Test-Path (Join-Path $WP_MysqlDir 'data\mysql'))) {
        if (-not (Initialize-WPMysqlData)) { return $false }
    }

    if (Test-WPPort 3306) {
        Write-WPLog '端口 3306 被占用，启动失败' 'ERROR'; return $false
    }

    $ini = Join-Path $WP_MysqlDir 'my.ini'
    Start-Process -FilePath $exe -ArgumentList "--defaults-file=`"$ini`"" `
        -WorkingDirectory $WP_MysqlDir -WindowStyle Hidden | Out-Null
    Start-Sleep -Seconds 2
    $ok = (Get-WPMysqlStatus).Running
    Write-WPLog ("MySQL 启动 " + $(if ($ok) {'成功'} else {'失败'}))
    return $ok
}

function Stop-WPMysql {
    $admin = Join-Path $WP_MysqlDir 'bin\mysqladmin.exe'
    if (Test-Path $admin) {
        $state = Get-WPState
        # 尝试优雅关停
        try {
            & $admin -u root --protocol=tcp -h 127.0.0.1 shutdown 2>&1 | Out-Null
        } catch {}
    }
    Start-Sleep -Milliseconds 800
    Get-WPProcess -Name 'mysqld' -PathFilter $WP_MysqlDir | ForEach-Object {
        try { Stop-Process -Id $_.Id -Force } catch {}
    }
    Write-WPLog 'MySQL 已停止'
    return $true
}

function Restart-WPMysql {
    Stop-WPMysql | Out-Null
    Start-Sleep -Milliseconds 500
    return Start-WPMysql
}

function Set-WPMysqlRootPassword {
    param([string]$NewPassword)
    $admin = Join-Path $WP_MysqlDir 'bin\mysqladmin.exe'
    if (-not (Test-Path $admin)) { return $false }
    if (-not (Get-WPMysqlStatus).Running) { Write-WPLog 'MySQL 未运行' 'ERROR'; return $false }

    # 尝试空密码登录修改
    & $admin -u root --protocol=tcp -h 127.0.0.1 password $NewPassword 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-WPLog 'MySQL root 密码已修改'
        return $true
    }
    Write-WPLog 'MySQL 密码修改失败（如已设置过密码请手动操作）' 'ERROR'
    return $false
}
