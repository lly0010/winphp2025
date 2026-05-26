# Sites.ps1 - 网站 / vhost / hosts 文件管理

function Add-WPSite {
    param(
        [string]$Name,         # 站点别名 (生成 vhost 文件名)
        [string]$ServerName,   # nginx server_name (域名)
        [string]$Root,         # 网站根目录
        [int]$Port = 80,
        [bool]$AddHosts = $true
    )
    if ([string]::IsNullOrWhiteSpace($Name) -or [string]::IsNullOrWhiteSpace($ServerName)) {
        throw '站点名和域名不能为空'
    }
    if (-not (Test-Path $Root)) { New-Item -ItemType Directory $Root -Force | Out-Null }

    $vhostDir = Join-Path $WP_NginxDir 'conf\vhosts'
    if (-not (Test-Path $vhostDir)) { New-Item -ItemType Directory $vhostDir -Force | Out-Null }

    $tpl = Join-Path $WP_TplDir 'vhost.conf'
    $text = Expand-WPTemplate -TemplatePath $tpl -Tokens @{
        SITE        = $Name
        SERVER_NAME = $ServerName
        ROOT        = $Root.Replace('\','/')
        PORT        = "$Port"
    }
    $vhostFile = Join-Path $vhostDir "$Name.conf"
    Set-Content -Path $vhostFile -Value $text -Encoding ASCII

    # 默认欢迎页
    $idx = Join-Path $Root 'index.php'
    if (-not (Test-Path $idx) -and -not (Test-Path (Join-Path $Root 'index.html'))) {
        @"
<?php
echo "<h1>$ServerName</h1>";
echo "<p>This site is served by WinPHP.</p>";
echo "<p>PHP " . phpversion() . "</p>";
"@ | Out-File $idx -Encoding UTF8
    }

    # 保存到 sites.json
    $sites = @(Get-WPSites | Where-Object { $_.name -ne $Name })
    $sites += [pscustomobject]@{
        name       = $Name
        serverName = $ServerName
        root       = $Root
        port       = $Port
        createdAt  = (Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
    }
    Save-WPSites $sites

    if ($AddHosts -and $ServerName -ne 'localhost') {
        Add-WPHostsEntry -Domain $ServerName
    }

    Write-WPLog "已添加站点 $Name ($ServerName -> $Root)"
    Invoke-WPNginxReload | Out-Null
}

function Remove-WPSite {
    param([string]$Name, [bool]$RemoveHosts = $true)
    $sites = @(Get-WPSites)
    $target = $sites | Where-Object { $_.name -eq $Name } | Select-Object -First 1
    if (-not $target) { Write-WPLog "站点 $Name 不存在" 'WARN'; return }

    $vhostFile = Join-Path $WP_NginxDir "conf\vhosts\$Name.conf"
    if (Test-Path $vhostFile) { Remove-Item $vhostFile -Force }

    $remain = $sites | Where-Object { $_.name -ne $Name }
    Save-WPSites @($remain)

    if ($RemoveHosts -and $target.serverName -ne 'localhost') {
        Remove-WPHostsEntry -Domain $target.serverName
    }

    Write-WPLog "已删除站点 $Name"
    Invoke-WPNginxReload | Out-Null
}

# ---- hosts 文件管理 ----
$Global:WP_HostsTag = '# WinPHP'

function Add-WPHostsEntry {
    param([string]$Domain, [string]$IP = '127.0.0.1')
    if (-not (Test-Path $WP_HostsFile)) { return }
    try {
        $content = Get-Content $WP_HostsFile -ErrorAction Stop
    } catch {
        Write-WPLog "无法读取 hosts (需管理员): $_" 'ERROR'; return
    }
    $pattern = "^\s*[\d\.]+\s+$([regex]::Escape($Domain))(\s|$)"
    if ($content | Where-Object { $_ -match $pattern }) {
        Write-WPLog "hosts 已存在记录: $Domain"
        return
    }
    $line = "$IP`t$Domain`t$WP_HostsTag"
    try {
        Add-Content -Path $WP_HostsFile -Value $line -ErrorAction Stop
        Write-WPLog "hosts 已添加: $Domain"
    } catch {
        Write-WPLog "写入 hosts 失败 (需以管理员运行面板): $_" 'ERROR'
    }
}

function Remove-WPHostsEntry {
    param([string]$Domain)
    if (-not (Test-Path $WP_HostsFile)) { return }
    try {
        $content = Get-Content $WP_HostsFile -ErrorAction Stop
    } catch {
        Write-WPLog "无法读取 hosts: $_" 'ERROR'; return
    }
    $pattern = "^\s*[\d\.]+\s+$([regex]::Escape($Domain))(\s|$)"
    $new = $content | Where-Object { -not ($_ -match $pattern -and $_ -match [regex]::Escape($WP_HostsTag)) }
    try {
        Set-Content -Path $WP_HostsFile -Value $new -ErrorAction Stop
        Write-WPLog "hosts 已移除: $Domain"
    } catch {
        Write-WPLog "写入 hosts 失败 (需管理员): $_" 'ERROR'
    }
}
