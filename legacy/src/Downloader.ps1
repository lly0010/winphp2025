# Downloader.ps1 - 下载与解压组件

function Invoke-WPDownload {
    param(
        [string]$Url,
        [string]$OutFile,
        [scriptblock]$ProgressCallback = $null,
        [int]$MaxRetry = 3
    )
    $tmpDir = Split-Path $OutFile -Parent
    if (-not (Test-Path $tmpDir)) { New-Item -ItemType Directory $tmpDir -Force | Out-Null }

    # TLS 1.2/1.3 (有些 .NET 4.5 不支持 Tls13 枚举, 用值兜底)
    try {
        [Net.ServicePointManager]::SecurityProtocol = `
            [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]'Tls13'
    } catch {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    }

    $ua = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'

    $lastErr = $null
    for ($attempt = 1; $attempt -le $MaxRetry; $attempt++) {
        Write-WPLog "下载 (第 $attempt/$MaxRetry 次): $Url"
        try {
            $req = [System.Net.HttpWebRequest]::Create($Url)
            $req.UserAgent = $ua
            $req.Accept    = '*/*'
            $req.Headers.Add('Accept-Language', 'en-US,en;q=0.9,zh-CN;q=0.8')
            $req.AllowAutoRedirect = $true
            $req.Timeout       = 60000
            $req.ReadWriteTimeout = 60000
            $req.KeepAlive     = $false

            $resp   = $req.GetResponse()
            $total  = $resp.ContentLength
            $stream = $resp.GetResponseStream()
            $fs     = [System.IO.File]::Create($OutFile)
            $buffer = New-Object byte[] 65536
            $read   = 0L
            try {
                while (($n = $stream.Read($buffer, 0, $buffer.Length)) -gt 0) {
                    $fs.Write($buffer, 0, $n)
                    $read += $n
                    if ($ProgressCallback) { & $ProgressCallback $read $total }
                }
            } finally {
                $fs.Close(); $stream.Close(); $resp.Close()
            }
            Write-WPLog "下载完成: $OutFile ($([math]::Round($read/1MB, 2)) MB)"
            return
        } catch {
            $lastErr = $_
            Write-WPLog "下载失败 (第 $attempt 次): $_" 'WARN'
            if (Test-Path $OutFile) { Remove-Item $OutFile -Force -ErrorAction SilentlyContinue }
            if ($attempt -lt $MaxRetry) { Start-Sleep -Seconds ($attempt * 2) }
        }
    }
    throw "下载失败 (已重试 $MaxRetry 次): $lastErr"
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
        [ValidateSet('nginx','php','mysql','postgresql')]
        [string]$Type,
        [string]$Version,
        [scriptblock]$ProgressCallback = $null
    )
    $sources = Get-WPSources
    $entry = $sources.$Type | Where-Object { $_.version -eq $Version } | Select-Object -First 1
    if (-not $entry) { throw "未找到 $Type $Version 的下载源" }

    # 支持 urls 数组 (新), 也兼容 url 单字段 (旧)
    $urls = @()
    if ($entry.PSObject.Properties.Name -contains 'urls' -and $entry.urls) {
        $urls = @($entry.urls)
    }
    if (($urls.Count -eq 0) -and ($entry.PSObject.Properties.Name -contains 'url') -and $entry.url) {
        $urls = @($entry.url)
    }
    if ($urls.Count -eq 0) { throw "$Type $Version 没有任何下载 URL" }

    $tmpZip = Join-Path $WP_TmpDir ("$Type-$Version.zip")
    $downloaded = $false
    $lastErr = $null
    for ($i = 0; $i -lt $urls.Count; $i++) {
        $u = $urls[$i]
        Write-WPLog ("尝试下载源 {0}/{1}: {2}" -f ($i+1), $urls.Count, $u)
        try {
            Invoke-WPDownload -Url $u -OutFile $tmpZip -ProgressCallback $ProgressCallback -MaxRetry 2
            $downloaded = $true
            break
        } catch {
            $lastErr = $_
            Write-WPLog "下载源失败, 切换下一个: $_" 'WARN'
        }
    }
    if (-not $downloaded) {
        throw "$Type $Version 全部 $($urls.Count) 个下载源都失败. 最后错误: $lastErr"
    }

    $dest = switch ($Type) {
        'nginx'      { $WP_NginxDir }
        'php'        { $WP_PhpDir }
        'mysql'      { $WP_MysqlDir }
        'postgresql' { $WP_PgDir }
    }

    # 安装前先停止对应服务
    switch ($Type) {
        'nginx'      { Stop-WPNginx    | Out-Null }
        'php'        { Stop-WPPhp      | Out-Null }
        'mysql'      { Stop-WPMysql    | Out-Null }
        'postgresql' { Stop-WPPostgres | Out-Null }
    }

    Expand-WPZip -ZipPath $tmpZip -Destination $dest -RootInZip $entry.rootInZip
    Remove-Item $tmpZip -Force -ErrorAction SilentlyContinue

    # 安装后立即生成对应组件的配置
    switch ($Type) {
        'nginx'      { Initialize-WPNginxConfig }
        'php'        { Initialize-WPPhpConfig }
        'mysql'      { Initialize-WPMysqlConfig }
        'postgresql' { Initialize-WPPostgresConfig }
    }

    $state = Get-WPState
    switch ($Type) {
        'nginx'      { $state.nginxVersion = $Version }
        'php'        { $state.phpVersion   = $Version }
        'mysql'      { $state.mysqlVersion = $Version; $state.mysqlInited = $false }
        'postgresql' {
            if ($state.PSObject.Properties.Name -notcontains 'pgVersion') {
                $state | Add-Member -MemberType NoteProperty -Name pgVersion -Value '' -Force
            }
            $state.pgVersion = $Version
        }
    }
    Save-WPState $state
    Write-WPLog "$Type $Version 安装完成"
}

function Uninstall-WPComponent {
    param(
        [ValidateSet('nginx','php','mysql','postgresql')]
        [string]$Type,
        [bool]$KeepData = $false      # 对 mysql / postgresql 有意义: 保留 data 目录到 tmp/<type>-data-backup
    )

    # 先把对应的 Windows 服务卸掉 (若已注册)
    $svcName = switch ($Type) {
        'nginx'      { $WP_SvcNginx }
        'php'        { $WP_SvcPhp }
        'mysql'      { $WP_SvcMysql }
        'postgresql' { $WP_SvcPg }
    }
    if ($svcName -and (Get-Command Uninstall-WPService -ErrorAction SilentlyContinue)) {
        Uninstall-WPService -Name $svcName | Out-Null
    }

    # 停止直接进程
    switch ($Type) {
        'nginx'      { Stop-WPNginx    | Out-Null }
        'php'        { Stop-WPPhp      | Out-Null }
        'mysql'      { Stop-WPMysql    | Out-Null }
        'postgresql' { Stop-WPPostgres | Out-Null }
    }
    Start-Sleep -Milliseconds 800

    $dir = switch ($Type) {
        'nginx'      { $WP_NginxDir }
        'php'        { $WP_PhpDir }
        'mysql'      { $WP_MysqlDir }
        'postgresql' { $WP_PgDir }
    }

    # 数据库可选备份
    if ((($Type -eq 'mysql') -or ($Type -eq 'postgresql')) -and $KeepData) {
        $dataDir = Join-Path $dir 'data'
        if (Test-Path $dataDir) {
            $backup = Join-Path $WP_TmpDir ("$Type-data-backup-" + (Get-Date -Format 'yyyyMMdd-HHmmss'))
            Move-Item $dataDir $backup -Force
            Write-WPLog "$Type data 目录已备份: $backup"
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
        'nginx'      { $state.nginxVersion = '' }
        'php'        { $state.phpVersion   = '' }
        'mysql'      { $state.mysqlVersion = ''; $state.mysqlInited = $false }
        'postgresql' {
            if ($state.PSObject.Properties.Name -contains 'pgVersion') { $state.pgVersion = '' }
        }
    }
    Save-WPState $state
    Write-WPLog "$Type 已卸载"
    return $true
}
