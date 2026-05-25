# Downloader.ps1 - 下载与解压组件

function Invoke-WPDownload {
    param(
        [string]$Url,
        [string]$OutFile,
        [scriptblock]$ProgressCallback = $null
    )
    Write-WPLog "下载: $Url"
    $tmpDir = Split-Path $OutFile -Parent
    if (-not (Test-Path $tmpDir)) { New-Item -ItemType Directory $tmpDir -Force | Out-Null }

    # 启用 TLS 1.2，避免 Windows 10 旧默认拒绝部分官方源
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13

    $req = [System.Net.HttpWebRequest]::Create($Url)
    $req.UserAgent = 'WinPHP/1.0'
    $req.Timeout = 30000
    $resp = $req.GetResponse()
    $total = $resp.ContentLength
    $stream = $resp.GetResponseStream()
    $fs = [System.IO.File]::Create($OutFile)
    $buffer = New-Object byte[] 65536
    $read = 0L
    while (($n = $stream.Read($buffer, 0, $buffer.Length)) -gt 0) {
        $fs.Write($buffer, 0, $n)
        $read += $n
        if ($ProgressCallback) {
            & $ProgressCallback $read $total
        }
    }
    $fs.Close(); $stream.Close(); $resp.Close()
    Write-WPLog "下载完成: $OutFile ($([math]::Round($read/1MB, 2)) MB)"
}

function Expand-WPZip {
    param(
        [string]$ZipPath,
        [string]$Destination,
        [string]$RootInZip = ''
    )
    Write-WPLog "解压: $ZipPath -> $Destination"
    Add-Type -AssemblyName System.IO.Compression.FileSystem

    if (Test-Path $Destination) { Remove-Item $Destination -Recurse -Force }

    if ([string]::IsNullOrEmpty($RootInZip)) {
        [System.IO.Compression.ZipFile]::ExtractToDirectory($ZipPath, $Destination)
    } else {
        # 解压到临时目录，再将 RootInZip 内容剪切到 Destination
        $tmp = Join-Path $env:TEMP ("winphp_" + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory $tmp -Force | Out-Null
        try {
            [System.IO.Compression.ZipFile]::ExtractToDirectory($ZipPath, $tmp)
            $src = Join-Path $tmp $RootInZip
            if (-not (Test-Path $src)) {
                # 自动识别第一层目录
                $first = Get-ChildItem $tmp | Select-Object -First 1
                if ($first -and $first.PSIsContainer) { $src = $first.FullName }
            }
            Move-Item -Path $src -Destination $Destination -Force
        } finally {
            if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue }
        }
    }
    Write-WPLog "解压完成: $Destination"
}

function Install-WPComponent {
    param(
        [ValidateSet('nginx','php','mysql')]
        [string]$Type,
        [string]$Version,
        [scriptblock]$ProgressCallback = $null
    )
    $sources = Get-WPSources
    $entry = $sources.$Type | Where-Object { $_.version -eq $Version } | Select-Object -First 1
    if (-not $entry) { throw "未找到 $Type $Version 的下载源" }

    $tmpZip = Join-Path $WP_TmpDir ("$Type-$Version.zip")
    Invoke-WPDownload -Url $entry.url -OutFile $tmpZip -ProgressCallback $ProgressCallback

    $dest = switch ($Type) {
        'nginx' { $WP_NginxDir }
        'php'   { $WP_PhpDir }
        'mysql' { $WP_MysqlDir }
    }

    # 安装前先停止对应服务
    switch ($Type) {
        'nginx' { Stop-WPNginx | Out-Null }
        'php'   { Stop-WPPhp   | Out-Null }
        'mysql' { Stop-WPMysql | Out-Null }
    }

    Expand-WPZip -ZipPath $tmpZip -Destination $dest -RootInZip $entry.rootInZip
    Remove-Item $tmpZip -Force -ErrorAction SilentlyContinue

    # 安装后立即生成对应组件的配置
    switch ($Type) {
        'nginx' { Initialize-WPNginxConfig }
        'php'   { Initialize-WPPhpConfig }
        'mysql' { Initialize-WPMysqlConfig }
    }

    $state = Get-WPState
    switch ($Type) {
        'nginx' { $state.nginxVersion = $Version }
        'php'   { $state.phpVersion   = $Version }
        'mysql' { $state.mysqlVersion = $Version; $state.mysqlInited = $false }
    }
    Save-WPState $state
    Write-WPLog "$Type $Version 安装完成"
}

function Uninstall-WPComponent {
    param(
        [ValidateSet('nginx','php','mysql')]
        [string]$Type,
        [bool]$KeepData = $false      # 仅对 mysql 有意义: 保留 data 目录到 tmp/mysql-data-backup
    )

    # 先把对应的 Windows 服务卸掉 (若已注册)
    $svcName = switch ($Type) {
        'nginx' { $WP_SvcNginx }
        'php'   { $WP_SvcPhp }
        'mysql' { $WP_SvcMysql }
    }
    if ($svcName -and (Get-Command Uninstall-WPService -ErrorAction SilentlyContinue)) {
        Uninstall-WPService -Name $svcName | Out-Null
    }

    # 停止直接进程
    switch ($Type) {
        'nginx' { Stop-WPNginx | Out-Null }
        'php'   { Stop-WPPhp   | Out-Null }
        'mysql' { Stop-WPMysql | Out-Null }
    }
    Start-Sleep -Milliseconds 800

    $dir = switch ($Type) {
        'nginx' { $WP_NginxDir }
        'php'   { $WP_PhpDir }
        'mysql' { $WP_MysqlDir }
    }

    # MySQL 数据可选备份
    if ($Type -eq 'mysql' -and $KeepData) {
        $dataDir = Join-Path $dir 'data'
        if (Test-Path $dataDir) {
            $backup = Join-Path $WP_TmpDir ("mysql-data-backup-" + (Get-Date -Format 'yyyyMMdd-HHmmss'))
            Move-Item $dataDir $backup -Force
            Write-WPLog "MySQL data 目录已备份: $backup"
        }
    }

    if (Test-Path $dir) {
        try {
            Remove-Item $dir -Recurse -Force -ErrorAction Stop
            Write-WPLog "已删除 $dir"
        } catch {
            # 有时进程残留导致部分文件占用, 重试一次
            Start-Sleep -Seconds 1
            try {
                Remove-Item $dir -Recurse -Force -ErrorAction Stop
            } catch {
                Write-WPLog "删除目录失败,可能有文件被占用: $_" 'ERROR'
                throw
            }
        }
    }

    $state = Get-WPState
    switch ($Type) {
        'nginx' { $state.nginxVersion = '' }
        'php'   { $state.phpVersion   = '' }
        'mysql' { $state.mysqlVersion = ''; $state.mysqlInited = $false }
    }
    Save-WPState $state
    Write-WPLog "$Type 已卸载"
    return $true
}
