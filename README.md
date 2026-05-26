# WinPHP 2025

类似 phpStudy / FlyEnv 的 **Windows 一键部署面板**, 用 **Go + Wails + Vue 3** 重写, 单文件 EXE ~10 MB, 启动毫秒级.

一键管理 **Nginx + PHP + MySQL + PostgreSQL** 四件套, 支持多版本下载/切换, 网站管理, 数据库管理, PHP 扩展管理, 开机自启 (NSSM 服务化), 全部带图形界面.

## 功能

| 模块         | 内容                                                                                          |
|--------------|-----------------------------------------------------------------------------------------------|
| **首页**     | 4 张服务卡片 (状态/版本/端口/服务) + 快速操作 + 我的网站表格                                  |
| **服务控制** | Nginx, PHP-CGI, MySQL, PostgreSQL — 启停/重启/重载, 状态毫秒级刷新                            |
| **多版本**   | 各组件 3-7 个版本可选, 多 URL 自动回退 (官方 / archives 镜像)                                 |
| **网站**     | 图形化新建 vhost, 模板: PHP / Laravel / WordPress / 静态, 自动写 hosts, 一键 nginx reload     |
| **数据库**   | MySQL 修改 root 密码, PostgreSQL 信息, 一键命令行                                              |
| **PHP 扩展** | 列出 ext/ 下所有 DLL, 复选框启用/禁用, 一键重启 PHP-CGI 生效                                  |
| **开机自启** | NSSM 自动下载, 一键启用全部 (3 服务 + 面板任务计划), 也可手动指定 nssm.exe                    |
| **配置编辑** | nginx.conf / php.ini / my.ini / postgresql.conf / vhost 内置编辑器                            |
| **工具**     | 端口检测, hosts 编辑器, 一键打开各目录, 浏览器跳转, phpinfo                                   |
| **日志**     | 实时日志面板 + 级别筛选 + 关键字搜索 + 自动跟随                                                |

## 系统要求

- **Windows 10 / 11 / Server 2019+** (64位)
- **Edge WebView2 Runtime** — Win10/11 已自带, Server 可能要装 ([下载链接](https://developer.microsoft.com/microsoft-edge/webview2/))
- 首次安装组件时联网访问 `nginx.org` / `windows.php.net` / `dev.mysql.com` / `enterprisedb.com`

无需 .NET / Java / Python / PowerShell.

## 安装使用

### 方式 1: 下载已编译 EXE (推荐)

到 [Releases](../../releases) 页下载最新 `WinPHP.exe`, 放到任意空目录, 右键 → 以管理员身份运行.

### 方式 2: 自己编译

需要 Go 1.22+, Node 20+:

```bash
# 1. 装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. 装前端依赖
cd frontend && npm install && cd ..

# 3. 构建 (Release)
wails build -platform windows/amd64 -ldflags "-s -w" -trimpath

# 4. 产物在 build/bin/WinPHP.exe
```

### 方式 3: 开发模式 (热重载)

```bash
wails dev
```

打开 Vite 前端 + Go 后端, 改 Vue/CSS 即时生效, 改 Go 也会自动重启.

## 入门四步

1. 双击 `WinPHP.exe` (UAC 提示请同意 — 修改 hosts / 80 端口 / 注册服务都需要管理员)
2. 在首页 4 张服务卡片各点 **"安装 / 切换版本"** 选版本下载 (首次需联网)
3. 点顶部侧栏 **"全部启动"**, 或在卡片上单独启动
4. 浏览器访问 [http://localhost](http://localhost) 验证 → 看到欢迎页 + 点 `phpinfo` 链接

## 默认账号 (重要!)

| 数据库         | 端口 | 默认用户   | 默认密码        | 说明                                      |
|----------------|------|------------|-----------------|-------------------------------------------|
| **MySQL**      | 3306 | `root`     | (空)            | 首次启动用 `--initialize-insecure` 初始化 |
| **PostgreSQL** | 5432 | `postgres` | (空, trust 认证) | 首次启动用 `initdb --auth=trust` 初始化   |

**生产环境务必改密码**:
- MySQL: 在 "数据库" 页点 "修改 root 密码", 或 `mysqladmin -u root password "新密码"`
- PostgreSQL: 在 psql 里 `ALTER USER postgres WITH PASSWORD '新密码';`, 然后把 `bin/postgresql/data/pg_hba.conf` 的 `trust` 改成 `scram-sha-256` 并重启

PHP 里连接示例:
```php
$mysql = new PDO('mysql:host=127.0.0.1;dbname=mydb;charset=utf8mb4', 'root', '');
$pg    = new PDO('pgsql:host=127.0.0.1;port=5432;dbname=postgres', 'postgres', '');
```

## 新建网站 (类似 FlyEnv)

切到 "网站" → "+ 新建网站":
- 填站点名 (例如 `myblog`) → 域名 / 根目录自动建议 `myblog.local` / `www/myblog`
- 选网站类型: 普通 PHP / **Laravel** (root 自动指向 `public/`) / WordPress / 纯静态
- 勾 "同时创建 MySQL 数据库" 自动建库 (默认 root 空密码)
- 勾 "自动写入 hosts" 自动加 `127.0.0.1 myblog.local`
- 创建 → 自动 nginx reload → 浏览器访问 [http://myblog.local](http://myblog.local)

## 开机自启 (NSSM)

切到 "自启动" → "✓ 一键启用全部":
- NSSM 从 nssm.cc 自动下载 (~200KB) 到 `bin/nssm.exe`
- 把 Nginx / PHP-CGI / MySQL / PostgreSQL 注册成 Windows 服务 (启动类型: 自动)
- 把面板自身用任务计划程序 (登录时, 最高权限) 注册自启

如 nssm.cc 在国内访问失败 (503), 自己浏览器下 zip → 解压找 `win64\nssm.exe` → 点 "手动指定 nssm.exe..."

## PHP 扩展管理 (FlyEnv 风格)

切到 "PHP 扩展": 列出 `bin/php/ext/` 下所有 `php_*.dll`, 复选框启用/禁用 → 点 "应用" 自动重启 PHP-CGI 生效.

## 目录结构

```
WinPHP.exe                  # 单文件可执行 (双击运行)
bin/
├── nginx/                  # 安装后生成
├── php/
│   ├── ext/                # PHP 扩展 DLL
│   └── php.ini
├── mysql/
│   └── data/               # MySQL 数据 (注意备份)
├── postgresql/
│   └── data/               # PG 数据
└── nssm.exe                # 自启时下载
config/
├── sources.json            # 下载源 (可编辑覆盖内嵌默认)
├── state.json              # 已安装版本记录
└── sites.json              # 网站列表
www/
├── default/                # 默认欢迎页
└── <你的站点>/             # 用户创建的站点目录
logs/winphp.log             # 面板日志
tmp/                        # 下载临时目录
```

整个目录可整体拷到 U 盘移动. 但 Windows 服务注册的路径是绝对路径, 换路径前要先到面板里 "禁用全部开机自启".

## 体积 & 性能

| 指标       | 数值                                                                |
|------------|---------------------------------------------------------------------|
| EXE 大小   | ~10 MB (UPX 压缩后更小, 默认未启用)                                 |
| 内存占用   | ~50 MB (面板自身)                                                   |
| 启动时间   | <300ms (PowerShell 版本约 2-3s)                                     |
| 状态轮询   | 800ms 一次, 用 Win32 ToolHelp32 + 端口探测, 单次 <5ms               |
| WebView2   | 由 Edge 进程托管, 与系统共用                                        |

旧 PowerShell 版本性能差的原因: 每次轮询都启动 `Get-Process` + iterate MainModule, 单次几百毫秒; PowerShell 启动慢; WinForms 老旧重绘. 都已不复存在.

## 常见问题

**Q: 双击 EXE 无反应 / 闪一下?**
A: 检查 Windows 是否安装了 Edge WebView2. 命令行运行 `WinPHP.exe` 看错误.

**Q: 安装 PHP 报 404?**
A: PHP 老版本会从 `/releases/` 迁移到 `/releases/archives/`, sources.json 已配置两个 URL 自动回退. 如仍 404 说明该版本已被彻底撤回, 选其他版本.

**Q: NSSM 下载失败 (503)?**
A: nssm.cc 国内访问不稳, 自己挂代理下完后用 "手动指定 nssm.exe..." 选择本地文件.

**Q: PostgreSQL 启动失败?**
A: PG 在 Windows 上拒绝以管理员身份运行. 面板已用 `pg_ctl` 自动降权处理. 如仍失败看 `bin/postgresql/logs/postgres.log`.

**Q: 80 端口被占用?**
A: 工具页 → 检测端口 80. 常见是 IIS, Skype, 其他 web 服务. 关闭它们或改 nginx.conf 端口.

## 旧版本 (PowerShell + WinForms)

旧版代码保留在 [`legacy/`](legacy/) 目录, 可直接 `cd legacy && WinPHP.bat` 运行, 但不再维护. 性能远不如新版.

## License

MIT
