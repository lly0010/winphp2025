# Common.ps1 - 公共函数: 路径、配置、日志
# 由 WinPHP.ps1 在启动时 dot-source 加载

# ---- 全局路径 ----
$Global:WP_Root      = (Get-Item $PSScriptRoot).Parent.FullName
$Global:WP_BinDir    = Join-Path $WP_Root 'bin'
$Global:WP_NginxDir  = Join-Path $WP_BinDir 'nginx'
$Global:WP_PhpDir    = Join-Path $WP_BinDir 'php'
$Global:WP_MysqlDir  = Join-Path $WP_BinDir 'mysql'
$Global:WP_PgDir     = Join-Path $WP_BinDir 'postgresql'
$Global:WP_WwwDir    = Join-Path $WP_Root 'www'
$Global:WP_LogsDir   = Join-Path $WP_Root 'logs'
$Global:WP_TmpDir    = Join-Path $WP_Root 'tmp'
$Global:WP_ConfigDir = Join-Path $WP_Root 'config'
$Global:WP_TplDir    = Join-Path $WP_ConfigDir 'templates'
$Global:WP_SitesFile = Join-Path $WP_ConfigDir 'sites.json'
$Global:WP_StateFile = Join-Path $WP_ConfigDir 'state.json'

# 系统 hosts 文件
$Global:WP_HostsFile = "$env:SystemRoot\System32\drivers\etc\hosts"

# ---- 初始化目录 ----
function Initialize-WPDirs {
    foreach ($d in @($WP_BinDir, $WP_WwwDir, $WP_LogsDir, $WP_TmpDir,
                     (Join-Path $WP_WwwDir 'default'))) {
        if (-not (Test-Path $d)) { New-Item -ItemType Directory -Path $d -Force | Out-Null }
    }
}

# 兼容 PowerShell 5.x 没有 PlaceholderText 属性的情况 (TextBox.PlaceholderText 是 .NET 4.8+)
# 这里只是占位, 主要靠 .NET Framework 4.8 自身支持. 5.1 通常已经是 4.8.


# ---- 日志 ----
function Write-WPLog {
    param([string]$Message, [string]$Level = 'INFO')
    $line = "[{0}] [{1}] {2}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Level, $Message
    $logFile = Join-Path $WP_LogsDir 'winphp.log'
    try { Add-Content -Path $logFile -Value $line -Encoding UTF8 } catch {}
    if ($Global:WP_LogBox -ne $null) {
        $Global:WP_LogBox.AppendText($line + [Environment]::NewLine)
        $Global:WP_LogBox.SelectionStart = $Global:WP_LogBox.Text.Length
        $Global:WP_LogBox.ScrollToCaret()
    }
}

# ---- 配置: 已安装组件状态 ----
function Get-WPState {
    if (Test-Path $WP_StateFile) {
        try { return Get-Content $WP_StateFile -Raw -Encoding UTF8 | ConvertFrom-Json } catch {}
    }
    return [pscustomobject]@{
        nginxVersion = ''
        phpVersion   = ''
        mysqlVersion = ''
        mysqlInited  = $false
        autoStart    = $false
    }
}

function Save-WPState {
    param($State)
    $State | ConvertTo-Json -Depth 5 | Out-File $WP_StateFile -Encoding UTF8
}

# ---- 配置: 网站列表 ----
function Get-WPSites {
    if (Test-Path $WP_SitesFile) {
        try { return @(Get-Content $WP_SitesFile -Raw -Encoding UTF8 | ConvertFrom-Json) } catch {}
    }
    return @()
}

function Save-WPSites {
    param($Sites)
    ,$Sites | ConvertTo-Json -Depth 5 | Out-File $WP_SitesFile -Encoding UTF8
}

# ---- 下载源清单 ----
function Get-WPSources {
    $f = Join-Path $WP_ConfigDir 'sources.json'
    return Get-Content $f -Raw -Encoding UTF8 | ConvertFrom-Json
}

# ---- 进程辅助 ----
function Get-WPProcess {
    param([string]$Name, [string]$PathFilter)
    @(Get-Process -Name $Name -ErrorAction SilentlyContinue | Where-Object {
        try { $_.MainModule.FileName -like "$PathFilter*" } catch { $false }
    })
}

function Test-WPPort {
    param([int]$Port)
    try {
        $c = New-Object System.Net.Sockets.TcpClient
        $c.Connect('127.0.0.1', $Port)
        $c.Close()
        return $true
    } catch { return $false }
}

# ---- 模板渲染 ----
function Expand-WPTemplate {
    param(
        [string]$TemplatePath,
        [hashtable]$Tokens
    )
    $text = Get-Content $TemplatePath -Raw -Encoding UTF8
    foreach ($k in $Tokens.Keys) {
        $text = $text.Replace("##$k##", $Tokens[$k])
    }
    return $text
}

# ---- 文件锁、版本检测辅助 ----
function Get-NginxRunningVersion {
    $exe = Join-Path $WP_NginxDir 'nginx.exe'
    if (-not (Test-Path $exe)) { return '' }
    try {
        $v = & $exe -v 2>&1
        if ($v -match 'nginx/([\d\.]+)') { return $Matches[1] }
    } catch {}
    return ''
}

function Get-PHPInstalledVersion {
    $exe = Join-Path $WP_PhpDir 'php.exe'
    if (-not (Test-Path $exe)) { return '' }
    try {
        $v = & $exe -v 2>&1 | Select-Object -First 1
        if ($v -match 'PHP ([\d\.]+)') { return $Matches[1] }
    } catch {}
    return ''
}

function Get-MySQLInstalledVersion {
    $exe = Join-Path $WP_MysqlDir 'bin\mysqld.exe'
    if (-not (Test-Path $exe)) { return '' }
    try {
        $v = & $exe --version 2>&1
        if ($v -match 'Ver ([\d\.]+)') { return $Matches[1] }
    } catch {}
    return ''
}

function Get-PostgresInstalledVersion {
    $exe = Join-Path $WP_PgDir 'bin\postgres.exe'
    if (-not (Test-Path $exe)) { return '' }
    try {
        $v = & $exe --version 2>&1
        if ($v -match '([\d\.]+)') { return $Matches[1] }
    } catch {}
    return ''
}
