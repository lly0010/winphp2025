# WinPHP 2025

类似 phpStudy 的 Windows 一键部署面板, 用纯 PowerShell + WinForms 编写, 无需任何运行时, 双击 `.bat` 即可启动. 支持下载/切换多版本 Nginx、PHP、MySQL, 自动生成虚拟主机、自动写 hosts.

## 功能

- 一键下载 / 切换版本 (Nginx 1.24~1.27, PHP 7.4~8.3, MySQL 5.7/8.0)
- 服务启停管理: Nginx、PHP-CGI(FastCGI)、MySQL 三组按钮 + 顶部"全部启动/停止"
- 实时状态显示: 运行状态、版本号、端口占用
- 网站管理: 图形化新建 vhost, 自动写 hosts, 默认欢迎页, 一键浏览/打开目录
- 数据库工具: 修改 root 密码、打开 MySQL CLI、打开数据目录、一键重新初始化
- 配置编辑器: 内置 `nginx.conf` / `php.ini` / `my.ini` / vhost 文件编辑
- 工具盒: 浏览 localhost、phpinfo、端口检测、Nginx 语法测试、清空日志、编辑 hosts
- 日志面板: 实时显示面板操作日志
- 全部数据放在面板目录, 可整体迁移到 U 盘

## 系统要求

- Windows 10 / 11 / Server 2019+
- PowerShell 5.1 (Windows 自带, 无需安装)
- .NET Framework 4.5+ (Windows 自带)
- 第一次安装组件需要联网访问 `nginx.org`、`windows.php.net`、`dev.mysql.com`

## 使用方法

### 1. 启动面板

双击 `WinPHP.bat` (会自动请求管理员权限).
如果遇到错误想看详细信息, 改用 `WinPHP-Console.bat`.

### 2. 下载组件

打开后默认在"首页", 三个面板分别是 Nginx / PHP-CGI / MySQL.
点击各自的 **"安装 / 切换版本"** 按钮, 选择版本后下载. 下载大小:

| 组件   | 大小 (约) |
|--------|-----------|
| Nginx  | 2 MB      |
| PHP    | 30 MB     |
| MySQL  | 220 MB    |

### 3. 启动服务

回到首页, 点击 **"全部启动"**. 状态变成绿色 "● 运行中" 即可.
浏览器访问 [http://localhost](http://localhost) 看到欢迎页, 访问 `/phpinfo.php` 查看 PHP 信息.

### 4. 新建网站

切到"网站"标签 → "+ 新建站点":

- 站点名称: `myblog` (用作 vhost 文件名)
- 域名: `myblog.local`
- 端口: `80`
- 根目录: 自动填 `www/myblog`
- 勾选"自动写入 hosts" (需面板以管理员运行)

点"创建", Nginx 自动重载, hosts 自动添加 `127.0.0.1 myblog.local`.
浏览器访问 [http://myblog.local](http://myblog.local) 即可.

### 5. MySQL 第一次启动

首次启动 MySQL 会自动初始化 `data/` 目录, 耗时约 1~2 分钟, 默认 root 密码为空.
切到"数据库"标签 → "修改 root 密码" 改成你想要的.

## 目录结构

```
WinPHP/
├── WinPHP.bat              启动入口 (自动提升管理员)
├── WinPHP-Console.bat      调试启动 (显示控制台)
├── WinPHP.ps1              主 GUI 程序
├── src/
│   ├── Common.ps1          公共: 路径、日志、状态
│   ├── Downloader.ps1      下载、解压、安装组件
│   ├── Services.ps1        Nginx / PHP / MySQL 启停
│   └── Sites.ps1           vhost、hosts 文件管理
├── config/
│   ├── sources.json        各版本下载 URL 清单
│   ├── state.json          (运行后生成) 已安装版本记录
│   ├── sites.json          (运行后生成) 网站列表
│   └── templates/          配置模板
│       ├── nginx.conf
│       ├── vhost.conf
│       ├── php.ini
│       └── my.ini
├── scripts/
│   └── install-service.ps1 (可选) 注册为 Windows 服务实现开机自启
├── bin/                    (下载后生成) 三大组件二进制
│   ├── nginx/
│   ├── php/
│   └── mysql/
├── www/
│   └── default/            默认欢迎页
├── logs/                   面板自身日志
└── tmp/                    下载临时目录
```

## 开机自启 (可选)

面板默认以前台进程方式管理 Nginx/MySQL, 不会开机自启.
要让它们作为 Windows 服务运行:

1. 下载 [NSSM](https://nssm.cc/download), 解压把 `nssm.exe` 放到 `bin/` 目录
2. 以管理员身份运行 PowerShell, 执行:
   ```powershell
   cd C:\path\to\WinPHP
   .\scripts\install-service.ps1 -Action install
   ```
3. 卸载: `.\scripts\install-service.ps1 -Action uninstall`

## 常见问题

**Q: 提示"无法加载脚本"或执行策略错误?**
A: 使用 `WinPHP.bat` 启动 (内置 `-ExecutionPolicy Bypass`). 如果直接执行 `.ps1`, 先运行
`Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`.

**Q: 80 端口被占用?**
A: 常见是 IIS、Skype、其他 Web 服务. 用 "工具 → 检测端口占用 80" 确认, 然后停掉对应程序, 或修改 `nginx.conf` 改端口.

**Q: MySQL 启动后立即崩溃?**
A: 查看 `bin/mysql/logs/error.log`. 常见原因:
- 端口被占用 → 改 `my.ini` 的 `port`
- 已存在残留 `data/` 目录但与新版本不兼容 → "数据库" 标签 → "重新初始化数据"

**Q: hosts 文件写入失败?**
A: 必须以管理员身份运行面板. `WinPHP.bat` 已经自动请求, 如果手动执行 `.ps1` 则不会.

**Q: PHP 提示找不到扩展?**
A: 编辑 `bin/php/php.ini`, 检查 `extension_dir = "ext"` 这一行, 以及对应 `extension=xxx` 是否启用.

**Q: 想增加新版本?**
A: 编辑 `config/sources.json`, 按现有格式追加 `{version, url, rootInZip}` 即可, 面板自动识别.

## 安全提示

- 默认 Nginx 监听 `0.0.0.0:80`, MySQL 监听 `127.0.0.1:3306`. 仅作本地开发用, 不要直接对外暴露.
- 修改默认 MySQL root 密码.
- 不要把 `bin/mysql/data/` 提交到 Git 仓库.

## License

MIT
