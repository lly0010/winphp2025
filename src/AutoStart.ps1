# AutoStart.ps1 - NSSM 集成、Windows 服务安装/卸载、面板自启动
# 设计目标: 用户在"自启动"标签页点"启用全部", 一切搞定

# ---- 服务名约定 ----
$Global:WP_SvcNginx = 'WinPHPNginx'
$Global:WP_SvcPhp   = 'WinPHPPhp'
$Global:WP_SvcMysql = 'WinPHPMySQL'
$Global:WP_TaskName = 'WinPHPPanelAutoStart'

# ============================================================
# NSSM 管理 - 首次使用时自动下载
# ============================================================
function Get-WPNssmPath {
    $p = Join-Path $WP_BinDir 'nssm.exe'
    if (Test-Path $p) { return $p }
    return $null
}

function Install-WPNssm {
    param([scriptblock]$ProgressCallback = $null)

    $existing = Get-WPNssmPath
    if ($existing) { return $existing }

    $sources = Get-WPSources
    $nssmInfo = $sources.nssm
    if (-not $nssmInfo) { throw 'sources.json 缺少 nssm 配置' }

    Write-WPLog '首次启用自启动, 正在下载 NSSM...'
    $zip = Join-Path $WP_TmpDir 'nssm.zip'
    Invoke-WPDownload -Url $nssmInfo.url -OutFile $zip -ProgressCallback $ProgressCallback

    $tmpExtract = Join-Path $env:TEMP ("winphp_nssm_" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory $tmpExtract -Force | Out-Null
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zip, $tmpExtract)

        $src = Join-Path $tmpExtract $nssmInfo.exeInZip.Replace('/','\')
        if (-not (Test-Path $src)) {
            # 兜底: 在解压树里搜
            $src = Get-ChildItem $tmpExtract -Recurse -Filter 'nssm.exe' |
                   Where-Object { $_.FullName -match 'win64' } |
                   Select-Object -ExpandProperty FullName -First 1
            if (-not $src) { throw '在 NSSM 压缩包内未找到 win64/nssm.exe' }
        }
        $dst = Join-Path $WP_BinDir 'nssm.exe'
        Copy-Item $src $dst -Force
        Write-WPLog "NSSM 已安装: $dst"
        return $dst
    } finally {
        Remove-Item $tmpExtract -Recurse -Force -ErrorAction SilentlyContinue
        Remove-Item $zip -Force -ErrorAction SilentlyContinue
    }
}

# ============================================================
# Windows 服务: Nginx / PHP-CGI / MySQL
# ============================================================
function Get-WPServiceInfo {
    param([string]$Name)
    $svc = Get-Service -Name $Name -ErrorAction SilentlyContinue
    if (-not $svc) {
        return [pscustomobject]@{ Name = $Name; Installed = $false; Status = ''; StartType = '' }
    }
    return [pscustomobject]@{
        Name      = $Name
        Installed = $true
        Status    = "$($svc.Status)"
        StartType = "$($svc.StartType)"
    }
}

function Install-WPServiceNginx {
    $nssm = Install-WPNssm
    $exe = Join-Path $WP_NginxDir 'nginx.exe'
    if (-not (Test-Path $exe)) { Write-WPLog 'Nginx 未安装,无法注册服务' 'ERROR'; return $false }

    # 已存在则先移除
    if ((Get-WPServiceInfo $WP_SvcNginx).Installed) {
        & $nssm stop   $WP_SvcNginx 2>&1 | Out-Null
        & $nssm remove $WP_SvcNginx confirm 2>&1 | Out-Null
    }
    # 停止可能直接运行的进程
    Stop-WPNginx | Out-Null

    & $nssm install  $WP_SvcNginx $exe "-p" "$WP_NginxDir"
    & $nssm set      $WP_SvcNginx AppDirectory $WP_NginxDir
    & $nssm set      $WP_SvcNginx Start SERVICE_AUTO_START
    & $nssm set      $WP_SvcNginx Description 'WinPHP Nginx Web Server'
    & $nssm set      $WP_SvcNginx AppStdout (Join-Path $WP_NginxDir 'logs\nssm_stdout.log')
    & $nssm set      $WP_SvcNginx AppStderr (Join-Path $WP_NginxDir 'logs\nssm_stderr.log')
    Write-WPLog "服务 $WP_SvcNginx 已注册 (开机自启)"
    return $true
}

function Install-WPServicePhp {
    $nssm = Install-WPNssm
    $exe = Join-Path $WP_PhpDir 'php-cgi.exe'
    if (-not (Test-Path $exe)) { Write-WPLog 'PHP 未安装,无法注册服务' 'ERROR'; return $false }

    if ((Get-WPServiceInfo $WP_SvcPhp).Installed) {
        & $nssm stop   $WP_SvcPhp 2>&1 | Out-Null
        & $nssm remove $WP_SvcPhp confirm 2>&1 | Out-Null
    }
    Stop-WPPhp | Out-Null

    $ini = Join-Path $WP_PhpDir 'php.ini'
    & $nssm install  $WP_SvcPhp $exe "-b" "127.0.0.1:9000" "-c" "$ini"
    & $nssm set      $WP_SvcPhp AppDirectory $WP_PhpDir
    & $nssm set      $WP_SvcPhp AppEnvironmentExtra "PHP_FCGI_CHILDREN=5" "PHP_FCGI_MAX_REQUESTS=1000"
    & $nssm set      $WP_SvcPhp Start SERVICE_AUTO_START
    & $nssm set      $WP_SvcPhp Description 'WinPHP PHP-CGI (FastCGI) Service'
    & $nssm set      $WP_SvcPhp AppStdout (Join-Path $WP_PhpDir 'logs\nssm_stdout.log')
    & $nssm set      $WP_SvcPhp AppStderr (Join-Path $WP_PhpDir 'logs\nssm_stderr.log')
    Write-WPLog "服务 $WP_SvcPhp 已注册 (开机自启)"
    return $true
}

function Install-WPServiceMysql {
    $nssm = Install-WPNssm
    $exe = Join-Path $WP_MysqlDir 'bin\mysqld.exe'
    if (-not (Test-Path $exe)) { Write-WPLog 'MySQL 未安装,无法注册服务' 'ERROR'; return $false }

    if ((Get-WPServiceInfo $WP_SvcMysql).Installed) {
        & $nssm stop   $WP_SvcMysql 2>&1 | Out-Null
        & $nssm remove $WP_SvcMysql confirm 2>&1 | Out-Null
    }
    Stop-WPMysql | Out-Null

    # 服务启动前先确保 data 已初始化
    $state = Get-WPState
    if (-not $state.mysqlInited -or -not (Test-Path (Join-Path $WP_MysqlDir 'data\mysql'))) {
        Initialize-WPMysqlData | Out-Null
    }

    $ini = Join-Path $WP_MysqlDir 'my.ini'
    & $nssm install  $WP_SvcMysql $exe "--defaults-file=$ini"
    & $nssm set      $WP_SvcMysql AppDirectory $WP_MysqlDir
    & $nssm set      $WP_SvcMysql Start SERVICE_AUTO_START
    & $nssm set      $WP_SvcMysql Description 'WinPHP MySQL Database Server'
    & $nssm set      $WP_SvcMysql AppStdout (Join-Path $WP_MysqlDir 'logs\nssm_stdout.log')
    & $nssm set      $WP_SvcMysql AppStderr (Join-Path $WP_MysqlDir 'logs\nssm_stderr.log')
    Write-WPLog "服务 $WP_SvcMysql 已注册 (开机自启)"
    return $true
}

function Uninstall-WPService {
    param([string]$Name)
    if (-not (Get-WPServiceInfo $Name).Installed) { return $true }
    $nssm = Get-WPNssmPath
    if ($nssm) {
        & $nssm stop   $Name 2>&1 | Out-Null
        & $nssm remove $Name confirm 2>&1 | Out-Null
    } else {
        sc.exe stop   $Name 2>&1 | Out-Null
        sc.exe delete $Name 2>&1 | Out-Null
    }
    Start-Sleep -Milliseconds 500
    Write-WPLog "服务 $Name 已卸载"
    return $true
}

# ============================================================
# 面板自启动 - 任务计划程序 (登录时以最高权限运行, 无 UAC)
# ============================================================
function Get-WPPanelAutoStartStatus {
    try {
        $t = Get-ScheduledTask -TaskName $WP_TaskName -ErrorAction SilentlyContinue
        return [bool]$t
    } catch { return $false }
}

function Enable-WPPanelAutoStart {
    $bat = Join-Path $WP_Root 'WinPHP.bat'
    if (-not (Test-Path $bat)) { Write-WPLog 'WinPHP.bat 不存在' 'ERROR'; return $false }
    try {
        # 已存在先删
        if (Get-WPPanelAutoStartStatus) {
            Unregister-ScheduledTask -TaskName $WP_TaskName -Confirm:$false -ErrorAction SilentlyContinue
        }
        $action    = New-ScheduledTaskAction -Execute $bat -WorkingDirectory $WP_Root
        $trigger   = New-ScheduledTaskTrigger -AtLogOn
        $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Highest
        $settings  = New-ScheduledTaskSettingsSet `
                        -AllowStartIfOnBatteries `
                        -DontStopIfGoingOnBatteries `
                        -StartWhenAvailable `
                        -ExecutionTimeLimit ([TimeSpan]::Zero)
        Register-ScheduledTask -TaskName $WP_TaskName `
            -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null
        Write-WPLog "面板自启动已启用 (任务: $WP_TaskName)"
        return $true
    } catch {
        Write-WPLog "面板自启动失败: $_" 'ERROR'
        return $false
    }
}

function Disable-WPPanelAutoStart {
    try {
        if (Get-WPPanelAutoStartStatus) {
            Unregister-ScheduledTask -TaskName $WP_TaskName -Confirm:$false -ErrorAction Stop
            Write-WPLog '面板自启动已禁用'
        }
        return $true
    } catch {
        Write-WPLog "面板自启动禁用失败: $_" 'ERROR'
        return $false
    }
}

# ============================================================
# 一键 启用 / 禁用 全部自启动
# ============================================================
function Enable-WPAllAutoStart {
    Install-WPNssm | Out-Null

    $results = @{}
    if (Test-Path (Join-Path $WP_NginxDir 'nginx.exe'))         { $results.Nginx = Install-WPServiceNginx } else { $results.Nginx = $null }
    if (Test-Path (Join-Path $WP_PhpDir   'php-cgi.exe'))       { $results.Php   = Install-WPServicePhp   } else { $results.Php   = $null }
    if (Test-Path (Join-Path $WP_MysqlDir 'bin\mysqld.exe'))    { $results.Mysql = Install-WPServiceMysql } else { $results.Mysql = $null }
    $results.Panel = Enable-WPPanelAutoStart

    # 启动已注册的服务
    foreach ($n in @($WP_SvcNginx, $WP_SvcPhp, $WP_SvcMysql)) {
        if ((Get-WPServiceInfo $n).Installed) {
            try { Start-Service -Name $n -ErrorAction Stop } catch {}
        }
    }
    return $results
}

function Disable-WPAllAutoStart {
    Uninstall-WPService $WP_SvcNginx | Out-Null
    Uninstall-WPService $WP_SvcPhp   | Out-Null
    Uninstall-WPService $WP_SvcMysql | Out-Null
    Disable-WPPanelAutoStart | Out-Null
    return $true
}
