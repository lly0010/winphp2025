# WinPHP.ps1 - WinForms 主 GUI
# 类似 phpStudy 的 Windows PHP/MySQL/Nginx 一键部署面板

$ErrorActionPreference = 'Stop'

# 加载模块
$srcDir = Join-Path $PSScriptRoot 'src'
. (Join-Path $srcDir 'Common.ps1')
. (Join-Path $srcDir 'Downloader.ps1')
. (Join-Path $srcDir 'Services.ps1')
. (Join-Path $srcDir 'Sites.ps1')

Initialize-WPDirs

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

# ============ 主窗口 ============
$form = New-Object System.Windows.Forms.Form
$form.Text = 'WinPHP 2025 - PHP/MySQL/Nginx 一键部署面板'
$form.Size = New-Object System.Drawing.Size(960, 680)
$form.StartPosition = 'CenterScreen'
$form.MinimumSize = $form.Size
$form.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 9)

# ---- 顶部栏 ----
$header = New-Object System.Windows.Forms.Panel
$header.Dock = 'Top'
$header.Height = 56
$header.BackColor = [System.Drawing.Color]::FromArgb(45, 116, 184)
$form.Controls.Add($header)

$title = New-Object System.Windows.Forms.Label
$title.Text = 'WinPHP 2025'
$title.ForeColor = [System.Drawing.Color]::White
$title.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 14, [System.Drawing.FontStyle]::Bold)
$title.AutoSize = $true
$title.Location = New-Object System.Drawing.Point(18, 14)
$header.Controls.Add($title)

$subtitle = New-Object System.Windows.Forms.Label
$subtitle.Text = '一键部署 PHP + MySQL + Nginx'
$subtitle.ForeColor = [System.Drawing.Color]::White
$subtitle.AutoSize = $true
$subtitle.Location = New-Object System.Drawing.Point(180, 22)
$header.Controls.Add($subtitle)

# 顶部右侧的"全部启动 / 全部停止"按钮
function New-HeaderButton {
    param([string]$Text, [int]$X)
    $b = New-Object System.Windows.Forms.Button
    $b.Text = $Text
    $b.Size = New-Object System.Drawing.Size(96, 30)
    $b.Location = New-Object System.Drawing.Point($X, 13)
    $b.FlatStyle = 'Flat'
    $b.FlatAppearance.BorderColor = [System.Drawing.Color]::White
    $b.BackColor = [System.Drawing.Color]::FromArgb(60, 130, 200)
    $b.ForeColor = [System.Drawing.Color]::White
    $b.Anchor = 'Top,Right'
    return $b
}
$btnStartAll = New-HeaderButton -Text '全部启动' -X 720
$btnStopAll  = New-HeaderButton -Text '全部停止' -X 825
$header.Controls.AddRange(@($btnStartAll, $btnStopAll))

# ---- 底部状态栏 ----
$statusBar = New-Object System.Windows.Forms.StatusStrip
$statusLabel = New-Object System.Windows.Forms.ToolStripStatusLabel
$statusLabel.Text = '就绪'
$statusLabel.Spring = $true
$statusLabel.TextAlign = 'MiddleLeft'
$statusBar.Items.Add($statusLabel) | Out-Null
$form.Controls.Add($statusBar)

function Set-WPStatus([string]$msg) { $statusLabel.Text = $msg }

# ---- TabControl ----
$tabs = New-Object System.Windows.Forms.TabControl
$tabs.Dock = 'Fill'
$form.Controls.Add($tabs)

# ============================================================
# Tab 1: 首页 (服务状态)
# ============================================================
$tabHome = New-Object System.Windows.Forms.TabPage
$tabHome.Text = '  首页  '
$tabHome.Padding = New-Object System.Windows.Forms.Padding(12)
$tabs.TabPages.Add($tabHome)

function New-ServiceBox {
    param([string]$Name, [int]$X)
    $g = New-Object System.Windows.Forms.GroupBox
    $g.Text = " $Name "
    $g.Location = New-Object System.Drawing.Point($X, 12)
    $g.Size = New-Object System.Drawing.Size(290, 260)
    $g.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 10, [System.Drawing.FontStyle]::Bold)

    $statusLbl = New-Object System.Windows.Forms.Label
    $statusLbl.Name = 'lblStatus'
    $statusLbl.Text = '● 未运行'
    $statusLbl.ForeColor = [System.Drawing.Color]::Gray
    $statusLbl.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 12, [System.Drawing.FontStyle]::Bold)
    $statusLbl.Location = New-Object System.Drawing.Point(20, 35)
    $statusLbl.AutoSize = $true
    $g.Controls.Add($statusLbl)

    $verLbl = New-Object System.Windows.Forms.Label
    $verLbl.Name = 'lblVer'
    $verLbl.Text = '版本: 未安装'
    $verLbl.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 9)
    $verLbl.ForeColor = [System.Drawing.Color]::DimGray
    $verLbl.Location = New-Object System.Drawing.Point(20, 75)
    $verLbl.AutoSize = $true
    $g.Controls.Add($verLbl)

    $portLbl = New-Object System.Windows.Forms.Label
    $portLbl.Name = 'lblPort'
    $portLbl.Text = ''
    $portLbl.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 9)
    $portLbl.ForeColor = [System.Drawing.Color]::DimGray
    $portLbl.Location = New-Object System.Drawing.Point(20, 100)
    $portLbl.AutoSize = $true
    $g.Controls.Add($portLbl)

    $btnStart = New-Object System.Windows.Forms.Button
    $btnStart.Name = 'btnStart'
    $btnStart.Text = '启动'
    $btnStart.Size = New-Object System.Drawing.Size(80, 32)
    $btnStart.Location = New-Object System.Drawing.Point(20, 145)
    $g.Controls.Add($btnStart)

    $btnStop = New-Object System.Windows.Forms.Button
    $btnStop.Name = 'btnStop'
    $btnStop.Text = '停止'
    $btnStop.Size = New-Object System.Drawing.Size(80, 32)
    $btnStop.Location = New-Object System.Drawing.Point(105, 145)
    $g.Controls.Add($btnStop)

    $btnRestart = New-Object System.Windows.Forms.Button
    $btnRestart.Name = 'btnRestart'
    $btnRestart.Text = '重启'
    $btnRestart.Size = New-Object System.Drawing.Size(80, 32)
    $btnRestart.Location = New-Object System.Drawing.Point(190, 145)
    $g.Controls.Add($btnRestart)

    $btnInstall = New-Object System.Windows.Forms.Button
    $btnInstall.Name = 'btnInstall'
    $btnInstall.Text = '安装 / 切换版本'
    $btnInstall.Size = New-Object System.Drawing.Size(165, 30)
    $btnInstall.Location = New-Object System.Drawing.Point(20, 195)
    $g.Controls.Add($btnInstall)

    $btnConfig = New-Object System.Windows.Forms.Button
    $btnConfig.Name = 'btnConfig'
    $btnConfig.Text = '配置文件'
    $btnConfig.Size = New-Object System.Drawing.Size(80, 30)
    $btnConfig.Location = New-Object System.Drawing.Point(190, 195)
    $g.Controls.Add($btnConfig)

    return $g
}

$boxNginx = New-ServiceBox -Name 'Nginx' -X 12
$boxPhp   = New-ServiceBox -Name 'PHP-CGI' -X 314
$boxMysql = New-ServiceBox -Name 'MySQL' -X 616
$tabHome.Controls.AddRange(@($boxNginx, $boxPhp, $boxMysql))

# 提示卡片
$tip = New-Object System.Windows.Forms.GroupBox
$tip.Text = ' 提示 '
$tip.Location = New-Object System.Drawing.Point(12, 285)
$tip.Size = New-Object System.Drawing.Size(894, 310)
$tip.Anchor = 'Top,Left,Right,Bottom'
$tip.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 10, [System.Drawing.FontStyle]::Bold)

$tipText = New-Object System.Windows.Forms.RichTextBox
$tipText.Dock = 'Fill'
$tipText.ReadOnly = $true
$tipText.BorderStyle = 'None'
$tipText.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 9)
$tipText.Text = @'
欢迎使用 WinPHP 2025!

入门四步:
  1. 切到"PHP / Nginx / MySQL"任一标签页, 点击"安装 / 切换版本"下载组件 (首次需要联网)
  2. 回到本页, 点击对应"启动"按钮, 或顶部"全部启动"
  3. 浏览器访问 http://localhost 验证默认欢迎页
  4. 在"网站"标签页中添加自己的域名和目录

提示:
  • 建议以管理员身份运行面板, 否则无法自动写入 hosts 文件
  • 默认 MySQL root 密码为空, 安装后请在"MySQL"标签页修改
  • 全部数据(组件、网站、数据库)位于面板根目录, 可整体迁移
  • 日志在"日志"标签页查看, 服务日志在各组件目录的 logs/ 下
'@
$tip.Controls.Add($tipText)
$tabHome.Controls.Add($tip)

# ============================================================
# 通用: 组件安装对话框
# ============================================================
function Show-InstallDialog {
    param([string]$Type)

    $dlg = New-Object System.Windows.Forms.Form
    $dlg.Text = "安装 / 切换 $Type 版本"
    $dlg.Size = New-Object System.Drawing.Size(480, 280)
    $dlg.StartPosition = 'CenterParent'
    $dlg.FormBorderStyle = 'FixedDialog'
    $dlg.MaximizeBox = $false; $dlg.MinimizeBox = $false

    $lbl = New-Object System.Windows.Forms.Label
    $lbl.Text = "选择要安装的 $Type 版本:"
    $lbl.Location = New-Object System.Drawing.Point(20, 18); $lbl.AutoSize = $true
    $dlg.Controls.Add($lbl)

    $combo = New-Object System.Windows.Forms.ComboBox
    $combo.DropDownStyle = 'DropDownList'
    $combo.Location = New-Object System.Drawing.Point(20, 45)
    $combo.Size = New-Object System.Drawing.Size(425, 25)
    $sources = Get-WPSources
    foreach ($v in $sources.$Type) { [void]$combo.Items.Add($v.version) }
    if ($combo.Items.Count -gt 0) { $combo.SelectedIndex = 0 }
    $dlg.Controls.Add($combo)

    $progress = New-Object System.Windows.Forms.ProgressBar
    $progress.Location = New-Object System.Drawing.Point(20, 90)
    $progress.Size = New-Object System.Drawing.Size(425, 22)
    $progress.Minimum = 0; $progress.Maximum = 100
    $dlg.Controls.Add($progress)

    $statusLbl = New-Object System.Windows.Forms.Label
    $statusLbl.Text = '就绪'
    $statusLbl.Location = New-Object System.Drawing.Point(20, 118)
    $statusLbl.Size = New-Object System.Drawing.Size(425, 50)
    $dlg.Controls.Add($statusLbl)

    $btnGo = New-Object System.Windows.Forms.Button
    $btnGo.Text = '开始安装'
    $btnGo.Size = New-Object System.Drawing.Size(90, 32)
    $btnGo.Location = New-Object System.Drawing.Point(260, 195)
    $dlg.Controls.Add($btnGo)

    $btnClose = New-Object System.Windows.Forms.Button
    $btnClose.Text = '关闭'
    $btnClose.Size = New-Object System.Drawing.Size(90, 32)
    $btnClose.Location = New-Object System.Drawing.Point(355, 195)
    $btnClose.DialogResult = 'Cancel'
    $dlg.Controls.Add($btnClose)
    $dlg.CancelButton = $btnClose

    $btnGo.Add_Click({
        if ($combo.SelectedItem -eq $null) { return }
        $btnGo.Enabled = $false; $btnClose.Enabled = $false; $combo.Enabled = $false
        $version = $combo.SelectedItem.ToString()
        $statusLbl.Text = "正在下载 $Type $version ..."
        $dlg.Refresh()
        try {
            $cb = {
                param($read, $total)
                if ($total -gt 0) {
                    $pct = [int](($read / $total) * 100)
                    if ($pct -gt 100) { $pct = 100 }
                    $progress.Value = $pct
                    $statusLbl.Text = ("已下载 {0:N1} MB / {1:N1} MB ({2}%)" -f ($read/1MB), ($total/1MB), $pct)
                    [System.Windows.Forms.Application]::DoEvents()
                }
            }
            Install-WPComponent -Type $Type -Version $version -ProgressCallback $cb
            $statusLbl.Text = "$Type $version 安装完成!"
            [System.Windows.Forms.MessageBox]::Show("$Type $version 安装完成", '成功', 'OK', 'Information') | Out-Null
            $dlg.DialogResult = 'OK'; $dlg.Close()
        } catch {
            $statusLbl.Text = "失败: $_"
            [System.Windows.Forms.MessageBox]::Show("安装失败: $_", '错误', 'OK', 'Error') | Out-Null
        } finally {
            $btnGo.Enabled = $true; $btnClose.Enabled = $true; $combo.Enabled = $true
        }
    })

    $dlg.ShowDialog($form) | Out-Null
    Refresh-WPHomeStatus
}

# ============================================================
# 配置文件编辑器
# ============================================================
function Show-ConfigEditor {
    param([string]$FilePath, [string]$Title)
    if (-not (Test-Path $FilePath)) {
        [System.Windows.Forms.MessageBox]::Show("配置文件不存在: $FilePath`n请先安装对应组件", '提示', 'OK', 'Warning') | Out-Null
        return
    }
    $dlg = New-Object System.Windows.Forms.Form
    $dlg.Text = $Title
    $dlg.Size = New-Object System.Drawing.Size(820, 600)
    $dlg.StartPosition = 'CenterParent'

    $tb = New-Object System.Windows.Forms.TextBox
    $tb.Multiline = $true
    $tb.ScrollBars = 'Both'
    $tb.WordWrap = $false
    $tb.Font = New-Object System.Drawing.Font('Consolas', 10)
    $tb.Dock = 'Fill'
    $tb.Text = (Get-Content $FilePath -Raw)
    $dlg.Controls.Add($tb)

    $pnl = New-Object System.Windows.Forms.Panel
    $pnl.Dock = 'Bottom'; $pnl.Height = 45
    $btnSave = New-Object System.Windows.Forms.Button
    $btnSave.Text = '保存'; $btnSave.Size = New-Object System.Drawing.Size(80, 30)
    $btnSave.Location = New-Object System.Drawing.Point(620, 8)
    $btnCancel = New-Object System.Windows.Forms.Button
    $btnCancel.Text = '取消'; $btnCancel.Size = New-Object System.Drawing.Size(80, 30)
    $btnCancel.Location = New-Object System.Drawing.Point(710, 8)
    $btnCancel.DialogResult = 'Cancel'
    $pnl.Controls.AddRange(@($btnSave, $btnCancel))
    $dlg.Controls.Add($pnl)
    $dlg.CancelButton = $btnCancel

    $btnSave.Add_Click({
        try {
            Set-Content -Path $FilePath -Value $tb.Text -Encoding ASCII
            Write-WPLog "已保存 $FilePath"
            [System.Windows.Forms.MessageBox]::Show('已保存. 请重启对应服务以生效.', '完成', 'OK', 'Information') | Out-Null
            $dlg.DialogResult = 'OK'; $dlg.Close()
        } catch {
            [System.Windows.Forms.MessageBox]::Show("保存失败: $_", '错误', 'OK', 'Error') | Out-Null
        }
    })
    $dlg.ShowDialog($form) | Out-Null
}

# ============================================================
# 首页 - 服务状态刷新
# ============================================================
function Refresh-WPHomeStatus {
    $items = @(
        @{ Box = $boxNginx; Status = Get-WPNginxStatus; Port = 80 },
        @{ Box = $boxPhp;   Status = Get-WPPhpStatus;   Port = 9000 },
        @{ Box = $boxMysql; Status = Get-WPMysqlStatus; Port = 3306 }
    )
    foreach ($it in $items) {
        $lblS = $it.Box.Controls['lblStatus']
        $lblV = $it.Box.Controls['lblVer']
        $lblP = $it.Box.Controls['lblPort']
        if ($it.Status.Running) {
            $lblS.Text = '● 运行中'
            $lblS.ForeColor = [System.Drawing.Color]::FromArgb(60, 170, 60)
        } else {
            $lblS.Text = '● 未运行'
            $lblS.ForeColor = [System.Drawing.Color]::Gray
        }
        if ([string]::IsNullOrEmpty($it.Status.Version)) {
            $lblV.Text = '版本: 未安装'
        } else {
            $lblV.Text = "版本: $($it.Status.Version)"
        }
        $lblP.Text = "端口: $($it.Port)"
    }
}

# 首页按钮事件绑定
$boxNginx.Controls['btnStart'].Add_Click({ Set-WPStatus '正在启动 Nginx...'; Start-WPNginx | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxNginx.Controls['btnStop'].Add_Click({ Set-WPStatus '正在停止 Nginx...'; Stop-WPNginx | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxNginx.Controls['btnRestart'].Add_Click({ Set-WPStatus '正在重启 Nginx...'; Restart-WPNginx | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxNginx.Controls['btnInstall'].Add_Click({ Show-InstallDialog -Type 'nginx' })
$boxNginx.Controls['btnConfig'].Add_Click({ Show-ConfigEditor -FilePath (Join-Path $WP_NginxDir 'conf\nginx.conf') -Title 'nginx.conf' })

$boxPhp.Controls['btnStart'].Add_Click({ Set-WPStatus '正在启动 PHP-CGI...'; Start-WPPhp | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxPhp.Controls['btnStop'].Add_Click({ Set-WPStatus '正在停止 PHP-CGI...'; Stop-WPPhp | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxPhp.Controls['btnRestart'].Add_Click({ Set-WPStatus '正在重启 PHP-CGI...'; Restart-WPPhp | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxPhp.Controls['btnInstall'].Add_Click({ Show-InstallDialog -Type 'php' })
$boxPhp.Controls['btnConfig'].Add_Click({ Show-ConfigEditor -FilePath (Join-Path $WP_PhpDir 'php.ini') -Title 'php.ini' })

$boxMysql.Controls['btnStart'].Add_Click({ Set-WPStatus '正在启动 MySQL...'; Start-WPMysql | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxMysql.Controls['btnStop'].Add_Click({ Set-WPStatus '正在停止 MySQL...'; Stop-WPMysql | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxMysql.Controls['btnRestart'].Add_Click({ Set-WPStatus '正在重启 MySQL...'; Restart-WPMysql | Out-Null; Refresh-WPHomeStatus; Set-WPStatus '就绪' })
$boxMysql.Controls['btnInstall'].Add_Click({ Show-InstallDialog -Type 'mysql' })
$boxMysql.Controls['btnConfig'].Add_Click({ Show-ConfigEditor -FilePath (Join-Path $WP_MysqlDir 'my.ini') -Title 'my.ini' })

$btnStartAll.Add_Click({
    Set-WPStatus '正在启动全部服务...'
    Start-WPNginx | Out-Null
    Start-WPPhp   | Out-Null
    Start-WPMysql | Out-Null
    Refresh-WPHomeStatus
    Set-WPStatus '就绪'
})
$btnStopAll.Add_Click({
    Set-WPStatus '正在停止全部服务...'
    Stop-WPNginx | Out-Null
    Stop-WPPhp   | Out-Null
    Stop-WPMysql | Out-Null
    Refresh-WPHomeStatus
    Set-WPStatus '就绪'
})

# ============================================================
# Tab 2: 网站管理
# ============================================================
$tabSites = New-Object System.Windows.Forms.TabPage
$tabSites.Text = '  网站  '
$tabSites.Padding = New-Object System.Windows.Forms.Padding(12)
$tabs.TabPages.Add($tabSites)

$lvSites = New-Object System.Windows.Forms.ListView
$lvSites.View = 'Details'
$lvSites.FullRowSelect = $true
$lvSites.GridLines = $true
$lvSites.Dock = 'Fill'
$lvSites.Columns.Add('名称', 120) | Out-Null
$lvSites.Columns.Add('域名', 220) | Out-Null
$lvSites.Columns.Add('端口', 60)  | Out-Null
$lvSites.Columns.Add('根目录', 360) | Out-Null
$lvSites.Columns.Add('创建时间', 140) | Out-Null

$sitesPanel = New-Object System.Windows.Forms.Panel
$sitesPanel.Dock = 'Top'; $sitesPanel.Height = 50

$btnAddSite = New-Object System.Windows.Forms.Button
$btnAddSite.Text = '+ 新建站点'
$btnAddSite.Size = New-Object System.Drawing.Size(110, 32)
$btnAddSite.Location = New-Object System.Drawing.Point(0, 10)

$btnDelSite = New-Object System.Windows.Forms.Button
$btnDelSite.Text = '删除站点'
$btnDelSite.Size = New-Object System.Drawing.Size(100, 32)
$btnDelSite.Location = New-Object System.Drawing.Point(120, 10)

$btnOpenSite = New-Object System.Windows.Forms.Button
$btnOpenSite.Text = '浏览器打开'
$btnOpenSite.Size = New-Object System.Drawing.Size(110, 32)
$btnOpenSite.Location = New-Object System.Drawing.Point(230, 10)

$btnOpenDir = New-Object System.Windows.Forms.Button
$btnOpenDir.Text = '打开目录'
$btnOpenDir.Size = New-Object System.Drawing.Size(100, 32)
$btnOpenDir.Location = New-Object System.Drawing.Point(350, 10)

$btnEditVhost = New-Object System.Windows.Forms.Button
$btnEditVhost.Text = '编辑 vhost'
$btnEditVhost.Size = New-Object System.Drawing.Size(110, 32)
$btnEditVhost.Location = New-Object System.Drawing.Point(460, 10)

$btnReloadSites = New-Object System.Windows.Forms.Button
$btnReloadSites.Text = '重载 Nginx'
$btnReloadSites.Size = New-Object System.Drawing.Size(110, 32)
$btnReloadSites.Location = New-Object System.Drawing.Point(580, 10)

$sitesPanel.Controls.AddRange(@($btnAddSite, $btnDelSite, $btnOpenSite, $btnOpenDir, $btnEditVhost, $btnReloadSites))
$tabSites.Controls.Add($lvSites)
$tabSites.Controls.Add($sitesPanel)

function Refresh-WPSitesList {
    $lvSites.Items.Clear()
    foreach ($s in Get-WPSites) {
        $it = New-Object System.Windows.Forms.ListViewItem $s.name
        $it.SubItems.Add([string]$s.serverName) | Out-Null
        $it.SubItems.Add([string]$s.port) | Out-Null
        $it.SubItems.Add([string]$s.root) | Out-Null
        $it.SubItems.Add([string]$s.createdAt) | Out-Null
        $lvSites.Items.Add($it) | Out-Null
    }
}

function Show-AddSiteDialog {
    $dlg = New-Object System.Windows.Forms.Form
    $dlg.Text = '新建站点'
    $dlg.Size = New-Object System.Drawing.Size(500, 320)
    $dlg.StartPosition = 'CenterParent'
    $dlg.FormBorderStyle = 'FixedDialog'
    $dlg.MaximizeBox = $false

    function New-DlgLabel($t, $y) {
        $l = New-Object System.Windows.Forms.Label
        $l.Text = $t; $l.Location = New-Object System.Drawing.Point(20, $y); $l.AutoSize = $true
        return $l
    }
    function New-DlgInput($y, $w = 350) {
        $t = New-Object System.Windows.Forms.TextBox
        $t.Location = New-Object System.Drawing.Point(110, $y)
        $t.Size = New-Object System.Drawing.Size($w, 25)
        return $t
    }

    $dlg.Controls.Add((New-DlgLabel '站点名称:' 22))
    $tbName = New-DlgInput 20
    $dlg.Controls.Add($tbName)

    $dlg.Controls.Add((New-DlgLabel '域名:' 62))
    $tbServer = New-DlgInput 60
    $tbServer.Text = 'test.local'
    $dlg.Controls.Add($tbServer)

    $dlg.Controls.Add((New-DlgLabel '端口:' 102))
    $tbPort = New-DlgInput 100 80
    $tbPort.Text = '80'
    $dlg.Controls.Add($tbPort)

    $dlg.Controls.Add((New-DlgLabel '根目录:' 142))
    $tbRoot = New-DlgInput 140 270
    $dlg.Controls.Add($tbRoot)
    $btnBrowse = New-Object System.Windows.Forms.Button
    $btnBrowse.Text = '...'
    $btnBrowse.Size = New-Object System.Drawing.Size(70, 26)
    $btnBrowse.Location = New-Object System.Drawing.Point(390, 140)
    $dlg.Controls.Add($btnBrowse)
    $btnBrowse.Add_Click({
        $fbd = New-Object System.Windows.Forms.FolderBrowserDialog
        $fbd.SelectedPath = $WP_WwwDir
        if ($fbd.ShowDialog() -eq 'OK') { $tbRoot.Text = $fbd.SelectedPath }
    })

    $cbHosts = New-Object System.Windows.Forms.CheckBox
    $cbHosts.Text = '自动写入 hosts (需管理员)'
    $cbHosts.Checked = $true
    $cbHosts.Location = New-Object System.Drawing.Point(110, 180)
    $cbHosts.AutoSize = $true
    $dlg.Controls.Add($cbHosts)

    $tbName.Add_TextChanged({
        if ([string]::IsNullOrWhiteSpace($tbRoot.Text)) {
            $tbRoot.Text = Join-Path $WP_WwwDir $tbName.Text
        }
    })

    $btnOK = New-Object System.Windows.Forms.Button
    $btnOK.Text = '创建'
    $btnOK.Size = New-Object System.Drawing.Size(90, 32)
    $btnOK.Location = New-Object System.Drawing.Point(280, 230)
    $btnCancel = New-Object System.Windows.Forms.Button
    $btnCancel.Text = '取消'
    $btnCancel.Size = New-Object System.Drawing.Size(90, 32)
    $btnCancel.Location = New-Object System.Drawing.Point(375, 230)
    $btnCancel.DialogResult = 'Cancel'
    $dlg.Controls.AddRange(@($btnOK, $btnCancel))
    $dlg.CancelButton = $btnCancel

    $btnOK.Add_Click({
        try {
            $port = 80
            [int]::TryParse($tbPort.Text, [ref]$port) | Out-Null
            $root = if ([string]::IsNullOrWhiteSpace($tbRoot.Text)) { Join-Path $WP_WwwDir $tbName.Text } else { $tbRoot.Text }
            Add-WPSite -Name $tbName.Text -ServerName $tbServer.Text -Root $root -Port $port -AddHosts $cbHosts.Checked
            $dlg.DialogResult = 'OK'; $dlg.Close()
        } catch {
            [System.Windows.Forms.MessageBox]::Show("创建失败: $_", '错误', 'OK', 'Error') | Out-Null
        }
    })

    if ($dlg.ShowDialog($form) -eq 'OK') { Refresh-WPSitesList }
}

$btnAddSite.Add_Click({ Show-AddSiteDialog })
$btnDelSite.Add_Click({
    if ($lvSites.SelectedItems.Count -eq 0) { return }
    $name = $lvSites.SelectedItems[0].Text
    $r = [System.Windows.Forms.MessageBox]::Show("确认删除站点 '$name'? (网站根目录文件不会被删除)", '确认', 'YesNo', 'Question')
    if ($r -eq 'Yes') { Remove-WPSite -Name $name; Refresh-WPSitesList }
})
$btnOpenSite.Add_Click({
    if ($lvSites.SelectedItems.Count -eq 0) { return }
    $name = $lvSites.SelectedItems[0].Text
    $s = Get-WPSites | Where-Object { $_.name -eq $name } | Select-Object -First 1
    if ($s) {
        $url = "http://$($s.serverName)" + $(if ($s.port -ne 80) { ":$($s.port)" } else { '' })
        Start-Process $url
    }
})
$btnOpenDir.Add_Click({
    if ($lvSites.SelectedItems.Count -eq 0) { return }
    $name = $lvSites.SelectedItems[0].Text
    $s = Get-WPSites | Where-Object { $_.name -eq $name } | Select-Object -First 1
    if ($s -and (Test-Path $s.root)) { Start-Process explorer.exe $s.root }
})
$btnEditVhost.Add_Click({
    if ($lvSites.SelectedItems.Count -eq 0) { return }
    $name = $lvSites.SelectedItems[0].Text
    Show-ConfigEditor -FilePath (Join-Path $WP_NginxDir "conf\vhosts\$name.conf") -Title "vhost: $name"
})
$btnReloadSites.Add_Click({
    if (Invoke-WPNginxReload) { Set-WPStatus 'Nginx 已重载' } else { Set-WPStatus '重载失败, 请查看日志' }
})

# ============================================================
# Tab 3: 数据库
# ============================================================
$tabDb = New-Object System.Windows.Forms.TabPage
$tabDb.Text = '  数据库  '
$tabDb.Padding = New-Object System.Windows.Forms.Padding(12)
$tabs.TabPages.Add($tabDb)

$dbInfo = New-Object System.Windows.Forms.GroupBox
$dbInfo.Text = ' MySQL 信息 '
$dbInfo.Location = New-Object System.Drawing.Point(12, 12)
$dbInfo.Size = New-Object System.Drawing.Size(900, 130)
$dbInfo.Anchor = 'Top,Left,Right'
$dbInfo.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 10, [System.Drawing.FontStyle]::Bold)

$lblDb = New-Object System.Windows.Forms.Label
$lblDb.Location = New-Object System.Drawing.Point(20, 30)
$lblDb.Size = New-Object System.Drawing.Size(800, 90)
$lblDb.Font = New-Object System.Drawing.Font('Consolas', 10)
$dbInfo.Controls.Add($lblDb)
$tabDb.Controls.Add($dbInfo)

$dbAction = New-Object System.Windows.Forms.GroupBox
$dbAction.Text = ' 数据库操作 '
$dbAction.Location = New-Object System.Drawing.Point(12, 155)
$dbAction.Size = New-Object System.Drawing.Size(900, 200)
$dbAction.Anchor = 'Top,Left,Right'
$dbAction.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 10, [System.Drawing.FontStyle]::Bold)

$btnPwd = New-Object System.Windows.Forms.Button
$btnPwd.Text = '修改 root 密码'
$btnPwd.Size = New-Object System.Drawing.Size(150, 36)
$btnPwd.Location = New-Object System.Drawing.Point(20, 35)
$dbAction.Controls.Add($btnPwd)

$btnCli = New-Object System.Windows.Forms.Button
$btnCli.Text = '打开 MySQL 命令行'
$btnCli.Size = New-Object System.Drawing.Size(150, 36)
$btnCli.Location = New-Object System.Drawing.Point(180, 35)
$dbAction.Controls.Add($btnCli)

$btnDataDir = New-Object System.Windows.Forms.Button
$btnDataDir.Text = '打开数据目录'
$btnDataDir.Size = New-Object System.Drawing.Size(150, 36)
$btnDataDir.Location = New-Object System.Drawing.Point(340, 35)
$dbAction.Controls.Add($btnDataDir)

$btnReinit = New-Object System.Windows.Forms.Button
$btnReinit.Text = '重新初始化数据'
$btnReinit.Size = New-Object System.Drawing.Size(150, 36)
$btnReinit.Location = New-Object System.Drawing.Point(500, 35)
$dbAction.Controls.Add($btnReinit)

$lblTip = New-Object System.Windows.Forms.Label
$lblTip.Text = "首次安装 MySQL 后,启动时会自动初始化 data 目录,默认 root 密码为空,请尽快修改.`n如果忘记密码,可通过'重新初始化数据'重置(数据会丢失!)."
$lblTip.Location = New-Object System.Drawing.Point(20, 90)
$lblTip.Size = New-Object System.Drawing.Size(860, 60)
$lblTip.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 9)
$lblTip.ForeColor = [System.Drawing.Color]::DimGray
$dbAction.Controls.Add($lblTip)

$tabDb.Controls.Add($dbAction)

function Refresh-WPDbInfo {
    $st = Get-WPMysqlStatus
    $running = if ($st.Running) { '运行中' } else { '未运行' }
    $lblDb.Text = @"
状态:       $running
版本:       $(if ($st.Version) {$st.Version} else {'未安装'})
端口:       3306 (绑定 127.0.0.1)
数据目录:   $(Join-Path $WP_MysqlDir 'data')
配置文件:   $(Join-Path $WP_MysqlDir 'my.ini')
默认账号:   root (默认无密码)
"@
}

$btnPwd.Add_Click({
    if (-not (Get-WPMysqlStatus).Running) {
        [System.Windows.Forms.MessageBox]::Show('请先启动 MySQL','提示','OK','Warning') | Out-Null; return
    }
    $dlg = New-Object System.Windows.Forms.Form
    $dlg.Text = '修改 MySQL root 密码'
    $dlg.Size = New-Object System.Drawing.Size(380, 180)
    $dlg.StartPosition = 'CenterParent'
    $dlg.FormBorderStyle = 'FixedDialog'

    $l = New-Object System.Windows.Forms.Label
    $l.Text = '新密码 (从空密码改起, 已有密码请用 CLI 修改):'
    $l.Location = New-Object System.Drawing.Point(20, 20); $l.AutoSize = $true
    $dlg.Controls.Add($l)
    $tb = New-Object System.Windows.Forms.TextBox
    $tb.Location = New-Object System.Drawing.Point(20, 50)
    $tb.Size = New-Object System.Drawing.Size(330, 25)
    $tb.UseSystemPasswordChar = $true
    $dlg.Controls.Add($tb)
    $btnOk = New-Object System.Windows.Forms.Button
    $btnOk.Text = '确定'; $btnOk.Size = New-Object System.Drawing.Size(80, 30)
    $btnOk.Location = New-Object System.Drawing.Point(170, 95)
    $btnCl = New-Object System.Windows.Forms.Button
    $btnCl.Text = '取消'; $btnCl.Size = New-Object System.Drawing.Size(80, 30)
    $btnCl.Location = New-Object System.Drawing.Point(270, 95)
    $btnCl.DialogResult = 'Cancel'
    $dlg.Controls.AddRange(@($btnOk, $btnCl))
    $dlg.CancelButton = $btnCl

    $btnOk.Add_Click({
        if ($tb.Text.Length -lt 4) {
            [System.Windows.Forms.MessageBox]::Show('密码长度至少 4 位','提示','OK','Warning')|Out-Null; return
        }
        if (Set-WPMysqlRootPassword -NewPassword $tb.Text) {
            [System.Windows.Forms.MessageBox]::Show('修改成功','完成','OK','Information')|Out-Null
            $dlg.Close()
        } else {
            [System.Windows.Forms.MessageBox]::Show('修改失败,请查看日志','错误','OK','Error')|Out-Null
        }
    })
    $dlg.ShowDialog($form) | Out-Null
})

$btnCli.Add_Click({
    $mysqlExe = Join-Path $WP_MysqlDir 'bin\mysql.exe'
    if (-not (Test-Path $mysqlExe)) {
        [System.Windows.Forms.MessageBox]::Show('MySQL 未安装','提示','OK','Warning')|Out-Null; return
    }
    Start-Process cmd.exe -ArgumentList "/k","`"$mysqlExe`" -u root -p -h 127.0.0.1" -WorkingDirectory (Join-Path $WP_MysqlDir 'bin')
})

$btnDataDir.Add_Click({
    $d = Join-Path $WP_MysqlDir 'data'
    if (Test-Path $d) { Start-Process explorer.exe $d }
})

$btnReinit.Add_Click({
    $r = [System.Windows.Forms.MessageBox]::Show(
        "这将停止 MySQL 并删除全部数据! 是否继续?",
        '危险操作', 'YesNo', 'Warning')
    if ($r -ne 'Yes') { return }
    Stop-WPMysql | Out-Null
    $d = Join-Path $WP_MysqlDir 'data'
    if (Test-Path $d) { Remove-Item $d -Recurse -Force }
    $state = Get-WPState; $state.mysqlInited = $false; Save-WPState $state
    if (Initialize-WPMysqlData) {
        [System.Windows.Forms.MessageBox]::Show('已重新初始化','完成','OK','Information')|Out-Null
    }
    Refresh-WPDbInfo
})

# ============================================================
# Tab 4: 工具
# ============================================================
$tabTools = New-Object System.Windows.Forms.TabPage
$tabTools.Text = '  工具  '
$tabTools.Padding = New-Object System.Windows.Forms.Padding(12)
$tabs.TabPages.Add($tabTools)

function New-ToolButton {
    param([string]$Text, [int]$X, [int]$Y, [scriptblock]$OnClick)
    $b = New-Object System.Windows.Forms.Button
    $b.Text = $Text
    $b.Size = New-Object System.Drawing.Size(180, 50)
    $b.Location = New-Object System.Drawing.Point($X, $Y)
    $b.Add_Click($OnClick)
    return $b
}

$tabTools.Controls.AddRange(@(
    (New-ToolButton '打开 www 目录'      20  20  { Start-Process explorer.exe $WP_WwwDir }),
    (New-ToolButton '打开面板根目录'      210 20  { Start-Process explorer.exe $WP_Root }),
    (New-ToolButton '打开日志目录'        400 20  { Start-Process explorer.exe $WP_LogsDir }),
    (New-ToolButton '编辑 hosts 文件'     590 20  { Start-Process notepad.exe $WP_HostsFile }),

    (New-ToolButton '浏览 localhost'     20  90  { Start-Process 'http://localhost' }),
    (New-ToolButton '浏览 phpinfo'       210 90  { Start-Process 'http://localhost/phpinfo.php' }),
    (New-ToolButton '检测端口占用 80'     400 90  {
        if (Test-WPPort 80) {
            [System.Windows.Forms.MessageBox]::Show('80 端口已被占用','结果','OK','Warning')|Out-Null
        } else {
            [System.Windows.Forms.MessageBox]::Show('80 端口空闲','结果','OK','Information')|Out-Null
        }
    }),
    (New-ToolButton '检测端口占用 3306'   590 90  {
        if (Test-WPPort 3306) {
            [System.Windows.Forms.MessageBox]::Show('3306 端口已被占用','结果','OK','Warning')|Out-Null
        } else {
            [System.Windows.Forms.MessageBox]::Show('3306 端口空闲','结果','OK','Information')|Out-Null
        }
    }),

    (New-ToolButton 'Nginx 配置测试'     20  160 {
        $exe = Join-Path $WP_NginxDir 'nginx.exe'
        if (Test-Path $exe) {
            $o = & $exe -t -p $WP_NginxDir 2>&1
            [System.Windows.Forms.MessageBox]::Show(($o -join "`n"),'结果','OK','Information')|Out-Null
        }
    }),
    (New-ToolButton 'Nginx 重载'         210 160 { Invoke-WPNginxReload | Out-Null }),
    (New-ToolButton 'PHP 版本信息'       400 160 {
        $exe = Join-Path $WP_PhpDir 'php.exe'
        if (Test-Path $exe) {
            $o = & $exe -v 2>&1
            [System.Windows.Forms.MessageBox]::Show(($o -join "`n"),'PHP','OK','Information')|Out-Null
        }
    }),
    (New-ToolButton '清空 nginx 日志'     590 160 {
        Get-ChildItem (Join-Path $WP_NginxDir 'logs') -Filter *.log -ErrorAction SilentlyContinue |
            ForEach-Object { Clear-Content $_.FullName -ErrorAction SilentlyContinue }
        Set-WPStatus '已清空 nginx 日志'
    })
))

# ============================================================
# Tab 5: 日志
# ============================================================
$tabLog = New-Object System.Windows.Forms.TabPage
$tabLog.Text = '  日志  '
$tabLog.Padding = New-Object System.Windows.Forms.Padding(8)
$tabs.TabPages.Add($tabLog)

$logBox = New-Object System.Windows.Forms.TextBox
$logBox.Multiline = $true
$logBox.ScrollBars = 'Vertical'
$logBox.ReadOnly = $true
$logBox.Dock = 'Fill'
$logBox.Font = New-Object System.Drawing.Font('Consolas', 9)
$logBox.BackColor = [System.Drawing.Color]::FromArgb(30, 30, 30)
$logBox.ForeColor = [System.Drawing.Color]::LightGreen
$tabLog.Controls.Add($logBox)
$Global:WP_LogBox = $logBox

$logPanel = New-Object System.Windows.Forms.Panel
$logPanel.Dock = 'Bottom'; $logPanel.Height = 36
$btnLogClr = New-Object System.Windows.Forms.Button
$btnLogClr.Text = '清空显示'; $btnLogClr.Size = New-Object System.Drawing.Size(90, 28)
$btnLogClr.Location = New-Object System.Drawing.Point(0, 4)
$btnLogClr.Add_Click({ $logBox.Clear() })
$btnLogOpen = New-Object System.Windows.Forms.Button
$btnLogOpen.Text = '打开日志文件'; $btnLogOpen.Size = New-Object System.Drawing.Size(120, 28)
$btnLogOpen.Location = New-Object System.Drawing.Point(95, 4)
$btnLogOpen.Add_Click({
    $f = Join-Path $WP_LogsDir 'winphp.log'
    if (Test-Path $f) { Start-Process notepad.exe $f }
})
$logPanel.Controls.AddRange(@($btnLogClr, $btnLogOpen))
$tabLog.Controls.Add($logPanel)

# ============================================================
# 启动 - 定时刷新状态
# ============================================================
$timer = New-Object System.Windows.Forms.Timer
$timer.Interval = 3000
$timer.Add_Tick({
    try {
        Refresh-WPHomeStatus
        Refresh-WPDbInfo
    } catch {}
})
$timer.Start()

$form.Add_Shown({
    Refresh-WPHomeStatus
    Refresh-WPSitesList
    Refresh-WPDbInfo
    Write-WPLog "WinPHP 启动. 根目录: $WP_Root"
    if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-WPLog '当前非管理员模式. 自动写入 hosts 等功能将无法使用.' 'WARN'
    }
})

$form.Add_FormClosing({
    $timer.Stop()
})

[void]$form.ShowDialog()
