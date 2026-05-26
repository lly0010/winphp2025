package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lly0010/winphp2025/internal/autostart"
	"github.com/lly0010/winphp2025/internal/download"
	"github.com/lly0010/winphp2025/internal/extract"
	"github.com/lly0010/winphp2025/internal/hosts"
	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/proc"
	"github.com/lly0010/winphp2025/internal/services"
	"github.com/lly0010/winphp2025/internal/sites"
	"github.com/lly0010/winphp2025/internal/sources"
	"github.com/lly0010/winphp2025/internal/state"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	nginx services.Nginx
	php   services.PHP
	mysql services.MySQL
	pg    services.Postgres

	statusStopCh chan struct{}

	// 进行中下载的取消函数, key = kind (nginx/php/mysql/postgres/nssm)
	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc
}

func NewApp() *App {
	return &App{
		statusStopCh: make(chan struct{}),
		cancels:      make(map[string]context.CancelFunc),
	}
}

// 注册一个可取消的下载任务
func (a *App) registerCancel(key string) context.Context {
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.cancelMu.Lock()
	// 同 key 已有则先取消
	if old, ok := a.cancels[key]; ok {
		old()
	}
	a.cancels[key] = cancel
	a.cancelMu.Unlock()
	return ctx
}

func (a *App) clearCancel(key string) {
	a.cancelMu.Lock()
	delete(a.cancels, key)
	a.cancelMu.Unlock()
}

// CancelInstall 取消正在进行的下载/安装. 前端 "取消下载" 按钮调用.
func (a *App) CancelInstall(kind string) {
	a.cancelMu.Lock()
	if c, ok := a.cancels[kind]; ok {
		c()
	}
	a.cancelMu.Unlock()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Info("WinPHP 启动, 根目录: %s", paths.Root)

	// 订阅日志推送到前端
	logCh := logger.Subscribe()
	go func() {
		for entry := range logCh {
			wruntime.EventsEmit(ctx, "log", entry)
		}
	}()

	// 启动状态轮询 (500ms 一次, 端口检测+ToolHelp32 都很快, 不会卡)
	go func() {
		t := time.NewTicker(800 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-a.statusStopCh:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				wruntime.EventsEmit(ctx, "status", a.AllStatus())
			}
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	close(a.statusStopCh)
}

// ============ 状态 ============

type AllStatusResult struct {
	Nginx          services.Status `json:"nginx"`
	Php            services.Status `json:"php"`
	Mysql          services.Status `json:"mysql"`
	Postgres       services.Status `json:"postgres"`
	IsAdmin        bool            `json:"isAdmin"`
	PanelAutoStart bool            `json:"panelAutoStart"`
}

func (a *App) AllStatus() AllStatusResult {
	return AllStatusResult{
		Nginx:          a.nginx.Status(),
		Php:            a.php.Status(),
		Mysql:          a.mysql.Status(),
		Postgres:       a.pg.Status(),
		IsAdmin:        isAdmin(),
		PanelAutoStart: autostart.PanelAutoStartEnabled(),
	}
}

// ============ 服务控制 ============

func (a *App) StartService(name string) error {
	switch name {
	case "nginx":
		return a.nginx.Start()
	case "php":
		return a.php.Start()
	case "mysql":
		return a.mysql.Start()
	case "postgres":
		return a.pg.Start()
	}
	return fmt.Errorf("未知服务: %s", name)
}

func (a *App) StopService(name string) error {
	switch name {
	case "nginx":
		return a.nginx.Stop()
	case "php":
		return a.php.Stop()
	case "mysql":
		return a.mysql.Stop()
	case "postgres":
		return a.pg.Stop()
	}
	return fmt.Errorf("未知服务: %s", name)
}

func (a *App) RestartService(name string) error {
	switch name {
	case "nginx":
		return a.nginx.Restart()
	case "php":
		return a.php.Restart()
	case "mysql":
		return a.mysql.Restart()
	case "postgres":
		return a.pg.Restart()
	}
	return fmt.Errorf("未知服务: %s", name)
}

func (a *App) StartAll() error {
	_ = a.nginx.Start()
	_ = a.php.Start()
	_ = a.mysql.Start()
	_ = a.pg.Start()
	return nil
}

func (a *App) StopAll() error {
	_ = a.nginx.Stop()
	_ = a.php.Stop()
	_ = a.mysql.Stop()
	_ = a.pg.Stop()
	return nil
}

func (a *App) NginxReload() error { return a.nginx.Reload() }

// ============ 安装 / 卸载 ============

func (a *App) ListVersions(kind string) ([]sources.VersionEntry, error) {
	src, err := sources.Load()
	if err != nil {
		return nil, err
	}
	switch kind {
	case "nginx":
		return src.Nginx, nil
	case "php":
		return src.Php, nil
	case "mysql":
		return src.Mysql, nil
	case "postgresql", "postgres":
		return src.Postgresql, nil
	}
	return nil, fmt.Errorf("未知组件: %s", kind)
}

// InstallComponent 下载并安装组件. mirror 控制 URL 优先级:
//   "cn"          中国镜像优先 (默认)
//   "oversea"     海外官方优先
//   "cn-only"     只用中国镜像
//   "oversea-only" 只用海外官方
func (a *App) InstallComponent(kind, version, mirror string) error {
	src, err := sources.Load()
	if err != nil {
		return err
	}
	entry := src.Find(kind, version)
	if entry == nil {
		return fmt.Errorf("未找到 %s %s 的下载源", kind, version)
	}
	urls := entry.MergedURLs(mirror)
	if len(urls) == 0 {
		return fmt.Errorf("当前镜像偏好下没有可用的下载 URL")
	}

	tmpZip := filepath.Join(paths.TmpDir, fmt.Sprintf("%s-%s.zip", kind, version))
	prog := func(d, t int64) {
		wruntime.EventsEmit(a.ctx, "install:progress", map[string]any{
			"kind":    kind,
			"version": version,
			"loaded":  d,
			"total":   t,
		})
	}
	wruntime.EventsEmit(a.ctx, "install:start", map[string]any{"kind": kind, "version": version})

	ctx := a.registerCancel(kind)
	defer a.clearCancel(kind)

	if err := download.DownloadWithRetry(ctx, urls, tmpZip, prog, 2); err != nil {
		_ = os.Remove(tmpZip)
		if errors.Is(err, context.Canceled) {
			logger.Info("%s %s 下载已取消", kind, version)
			wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "canceled": true})
			return fmt.Errorf("已取消")
		}
		wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "error": err.Error()})
		return err
	}

	dest := destDir(kind)
	a.stopFor(kind)
	if err := extract.Zip(tmpZip, dest, entry.RootInZip); err != nil {
		wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "error": err.Error()})
		return err
	}
	_ = os.Remove(tmpZip)

	a.initConfigFor(kind)

	st := state.Load()
	switch kind {
	case "nginx":
		st.NginxVersion = version
	case "php":
		st.PhpVersion = version
	case "mysql":
		st.MysqlVersion = version
		st.MysqlInited = false
	case "postgresql", "postgres":
		st.PgVersion = version
		st.PgInited = false
	}
	_ = state.Save(st)

	logger.Info("%s %s 安装完成", kind, version)
	wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "success": true})
	return nil
}

// PreviewUrls 前端在用户选完版本+镜像偏好后, 调它预览实际下载 URL 顺序.
func (a *App) PreviewUrls(kind, version, mirror string) ([]string, error) {
	src, err := sources.Load()
	if err != nil {
		return nil, err
	}
	entry := src.Find(kind, version)
	if entry == nil {
		return nil, fmt.Errorf("未找到 %s %s", kind, version)
	}
	return entry.MergedURLs(mirror), nil
}

func (a *App) UninstallComponent(kind string, keepData bool) error {
	// 先卸服务
	switch kind {
	case "nginx":
		_ = autostart.UnregisterService(services.NginxServiceName)
		_ = a.nginx.Stop()
	case "php":
		_ = autostart.UnregisterService(services.PhpServiceName)
		_ = a.php.Stop()
	case "mysql":
		_ = autostart.UnregisterService(services.MysqlServiceName)
		_ = a.mysql.Stop()
	case "postgresql", "postgres":
		_ = autostart.UnregisterService(services.PostgresServiceName)
		_ = a.pg.Stop()
	}
	time.Sleep(800 * time.Millisecond)

	dir := destDir(kind)
	// data 备份
	if keepData && (kind == "mysql" || kind == "postgresql" || kind == "postgres") {
		data := filepath.Join(dir, "data")
		if _, err := os.Stat(data); err == nil {
			backup := filepath.Join(paths.TmpDir, fmt.Sprintf("%s-data-backup-%s", kind, time.Now().Format("20060102-150405")))
			_ = os.Rename(data, backup)
			logger.Info("%s data 已备份到: %s", kind, backup)
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		// 重试一次, 文件占用问题
		time.Sleep(1 * time.Second)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("删除 %s 失败: %w", dir, err)
		}
	}
	st := state.Load()
	switch kind {
	case "nginx":
		st.NginxVersion = ""
	case "php":
		st.PhpVersion = ""
	case "mysql":
		st.MysqlVersion = ""
		st.MysqlInited = false
	case "postgresql", "postgres":
		st.PgVersion = ""
		st.PgInited = false
	}
	_ = state.Save(st)
	logger.Info("%s 已卸载", kind)
	return nil
}

func destDir(kind string) string {
	switch kind {
	case "nginx":
		return paths.NginxDir
	case "php":
		return paths.PhpDir
	case "mysql":
		return paths.MysqlDir
	case "postgresql", "postgres":
		return paths.PgDir
	}
	return ""
}

func (a *App) stopFor(kind string) {
	switch kind {
	case "nginx":
		_ = a.nginx.Stop()
	case "php":
		_ = a.php.Stop()
	case "mysql":
		_ = a.mysql.Stop()
	case "postgresql", "postgres":
		_ = a.pg.Stop()
	}
}

func (a *App) initConfigFor(kind string) {
	switch kind {
	case "nginx":
		_ = a.nginx.InitConfig()
	case "php":
		_ = a.php.InitConfig()
	case "mysql":
		_ = a.mysql.InitConfig()
	case "postgresql", "postgres":
		_ = a.pg.InitConfig()
	}
}

// ============ 网站 ============

func (a *App) ListSites() []state.Site                  { return sites.List() }
func (a *App) AddSite(in sites.AddSiteInput) error      { return sites.Add(in) }
func (a *App) RemoveSite(name string, rmHosts bool) error { return sites.Remove(name, rmHosts) }

// ============ hosts ============

func (a *App) ReadHosts() (string, error) { return hosts.Read() }
func (a *App) WriteHosts(text string) error { return hosts.Write(text) }

// ============ 配置编辑 ============

type ConfigKey string

func (a *App) ReadConfig(key string) (string, error) {
	p := configPath(key)
	if p == "" {
		return "", fmt.Errorf("未知配置: %s", key)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *App) WriteConfig(key, text string) error {
	p := configPath(key)
	if p == "" {
		return fmt.Errorf("未知配置: %s", key)
	}
	return os.WriteFile(p, []byte(text), 0o644)
}

func configPath(key string) string {
	switch key {
	case "nginx":
		return filepath.Join(paths.NginxDir, "conf", "nginx.conf")
	case "php":
		return filepath.Join(paths.PhpDir, "php.ini")
	case "mysql":
		return filepath.Join(paths.MysqlDir, "my.ini")
	case "postgres", "postgresql":
		return filepath.Join(paths.PgDir, "data", "postgresql.conf")
	}
	if strings.HasPrefix(key, "vhost:") {
		name := strings.TrimPrefix(key, "vhost:")
		return filepath.Join(paths.NginxDir, "conf", "vhosts", name+".conf")
	}
	return ""
}

// ============ 数据库 ============

func (a *App) MysqlSetPassword(newPwd string) error  { return a.mysql.SetRootPassword(newPwd) }
func (a *App) MysqlCreateDb(name, pwd string) error  { return a.mysql.CreateDatabase(name, pwd) }

// ============ PHP 扩展 ============

func (a *App) PhpExtensions() []services.Extension       { return a.php.ListExtensions() }
func (a *App) PhpSetExtension(name string, enable bool) error {
	if err := a.php.SetExtension(name, enable); err != nil {
		return err
	}
	logger.Info("PHP 扩展 %s -> %v (重启 PHP-CGI 生效)", name, enable)
	return nil
}

// ============ 自启 ============

func (a *App) EnsureNssm(mirror string) error {
	prog := func(d, t int64) {
		wruntime.EventsEmit(a.ctx, "nssm:progress", map[string]any{"loaded": d, "total": t})
	}
	ctx := a.registerCancel("nssm")
	defer a.clearCancel("nssm")
	_, err := autostart.EnsureNssm(ctx, mirror, prog)
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("已取消")
	}
	return err
}

func (a *App) PickAndInstallNssm() error {
	selected, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择 nssm.exe",
		Filters: []wruntime.FileFilter{
			{DisplayName: "nssm.exe", Pattern: "nssm.exe"},
			{DisplayName: "All EXE", Pattern: "*.exe"},
		},
	})
	if err != nil {
		return err
	}
	if selected == "" {
		return nil
	}
	return autostart.SetNssmFromFile(selected)
}

type AutoStartItem struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	BinReady  bool   `json:"binReady"`
}

func (a *App) AutoStartList() []AutoStartItem {
	mk := func(key, lbl, svcName string, st services.Status) AutoStartItem {
		return AutoStartItem{
			Key:       key,
			Label:     lbl,
			Installed: services.ServiceExists(svcName),
			Running:   services.ServiceStatusStr(svcName) == "Running",
			BinReady:  st.BinInstalled,
		}
	}
	return []AutoStartItem{
		{Key: "panel", Label: "WinPHP 面板 (登录自启)", Installed: autostart.PanelAutoStartEnabled(), Running: true, BinReady: true},
		mk("nginx", "Nginx 服务", services.NginxServiceName, a.nginx.Status()),
		mk("php", "PHP-CGI 服务", services.PhpServiceName, a.php.Status()),
		mk("mysql", "MySQL 服务", services.MysqlServiceName, a.mysql.Status()),
		mk("postgres", "PostgreSQL 服务", services.PostgresServiceName, a.pg.Status()),
	}
}

func (a *App) EnableAutoStart(key string) error {
	if key == "panel" {
		exe, err := autostart.CurrentExe()
		if err != nil {
			return err
		}
		return autostart.EnablePanelAutoStart(exe)
	}
	if err := a.EnsureNssm("cn"); err != nil {
		return fmt.Errorf("NSSM 安装失败: %w (可在'自启动'页面点'手动指定 nssm.exe'选择本地文件)", err)
	}
	switch key {
	case "nginx":
		return autostart.RegisterService(services.NginxServiceName,
			a.nginx.ExePath(), []string{"-p", paths.NginxDir},
			paths.NginxDir, "WinPHP Nginx", nil)
	case "php":
		return autostart.RegisterService(services.PhpServiceName,
			a.php.CgiPath(), []string{"-b", "127.0.0.1:9000", "-c", a.php.IniPath()},
			paths.PhpDir, "WinPHP PHP-CGI",
			map[string]string{"PHP_FCGI_CHILDREN": "5", "PHP_FCGI_MAX_REQUESTS": "1000"})
	case "mysql":
		return autostart.RegisterService(services.MysqlServiceName,
			a.mysql.ExePath(), []string{"--defaults-file=" + a.mysql.IniPath()},
			paths.MysqlDir, "WinPHP MySQL", nil)
	case "postgres":
		return autostart.RegisterService(services.PostgresServiceName,
			a.pg.ExePath(), []string{"-D", a.pg.DataPath()},
			paths.PgDir, "WinPHP PostgreSQL", nil)
	}
	return fmt.Errorf("未知: %s", key)
}

func (a *App) DisableAutoStart(key string) error {
	switch key {
	case "panel":
		return autostart.DisablePanelAutoStart()
	case "nginx":
		return autostart.UnregisterService(services.NginxServiceName)
	case "php":
		return autostart.UnregisterService(services.PhpServiceName)
	case "mysql":
		return autostart.UnregisterService(services.MysqlServiceName)
	case "postgres":
		return autostart.UnregisterService(services.PostgresServiceName)
	}
	return fmt.Errorf("未知: %s", key)
}

func (a *App) EnableAllAutoStart() error {
	keys := []string{"panel", "nginx", "php", "mysql", "postgres"}
	for _, k := range keys {
		// 跳过未安装的组件
		if k != "panel" {
			st := a.statusOf(k)
			if !st.BinInstalled {
				continue
			}
		}
		_ = a.EnableAutoStart(k)
	}
	return nil
}

func (a *App) DisableAllAutoStart() error {
	for _, k := range []string{"panel", "nginx", "php", "mysql", "postgres"} {
		_ = a.DisableAutoStart(k)
	}
	return nil
}

func (a *App) statusOf(key string) services.Status {
	switch key {
	case "nginx":
		return a.nginx.Status()
	case "php":
		return a.php.Status()
	case "mysql":
		return a.mysql.Status()
	case "postgres":
		return a.pg.Status()
	}
	return services.Status{}
}

// ============ 工具 ============

func (a *App) OpenInBrowser(url string) {
	wruntime.BrowserOpenURL(a.ctx, url)
}

// ============ 日志 ============

func (a *App) LogTail(n int) []logger.Entry { return logger.Tail(n) }

// ============ 路径暴露 (用于前端"打开目录"按钮) ============

type PathsResult struct {
	Root      string `json:"root"`
	BinDir    string `json:"binDir"`
	WwwDir    string `json:"wwwDir"`
	LogsDir   string `json:"logsDir"`
	HostsFile string `json:"hostsFile"`
}

func (a *App) GetPaths() PathsResult {
	return PathsResult{
		Root:      paths.Root,
		BinDir:    paths.BinDir,
		WwwDir:    paths.WwwDir,
		LogsDir:   paths.LogsDir,
		HostsFile: paths.HostsFile,
	}
}

func (a *App) OpenFolder(p string) error {
	// 用 explorer 打开
	if _, err := os.Stat(p); err != nil {
		return err
	}
	return execOpen(p)
}

// ============ 端口检测 ============

func (a *App) PortInUse(port int) bool { return proc.PortListening(port) }
