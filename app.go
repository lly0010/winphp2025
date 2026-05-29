package main

import (
	"context"
	"encoding/base64"
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
	"github.com/lly0010/winphp2025/internal/nettest"
	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/portcheck"
	"github.com/lly0010/winphp2025/internal/proc"
	"github.com/lly0010/winphp2025/internal/services"
	"github.com/lly0010/winphp2025/internal/sites"
	"github.com/lly0010/winphp2025/internal/sources"
	"github.com/lly0010/winphp2025/internal/state"
	"github.com/lly0010/winphp2025/internal/theme"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	nginx services.Nginx
	php   services.PHP
	mysql services.MySQL
	pg    services.Postgres
	redis services.Redis

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

	// 首次启动写入主题示例 (themes/example/ 给开发者参考)
	theme.EnsureSampleTheme()

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
	Redis          services.Status `json:"redis"`
	IsAdmin        bool            `json:"isAdmin"`
	PanelAutoStart bool            `json:"panelAutoStart"`
}

func (a *App) AllStatus() AllStatusResult {
	return AllStatusResult{
		Nginx:          a.nginx.Status(),
		Php:            a.php.Status(),
		Mysql:          a.mysql.Status(),
		Postgres:       a.pg.Status(),
		Redis:          a.redis.Status(),
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
	case "redis":
		return a.redis.Start()
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
	case "redis":
		return a.redis.Stop()
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
	case "redis":
		return a.redis.Restart()
	}
	return fmt.Errorf("未知服务: %s", name)
}

func (a *App) StartAll() error {
	_ = a.nginx.Start()
	_ = a.php.Start()
	_ = a.mysql.Start()
	_ = a.pg.Start()
	_ = a.redis.Start()
	return nil
}

func (a *App) StopAll() error {
	_ = a.nginx.Stop()
	_ = a.php.Stop()
	_ = a.mysql.Stop()
	_ = a.pg.Stop()
	_ = a.redis.Stop()
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
	case "redis":
		return src.Redis, nil
	}
	return nil, fmt.Errorf("未知组件: %s", kind)
}

// InstallComponent 下载并安装组件 (按 sources.json 里的 urls 顺序尝试, 全失败才报错).
// 如果该版本是自定义且只有 LocalZip 字段, 则跳过下载直接用本地 zip.
func (a *App) InstallComponent(kind, version string) error {
	src, err := sources.Load()
	if err != nil {
		return err
	}
	entry := src.Find(kind, version)
	if entry == nil {
		return fmt.Errorf("未找到 %s %s 的下载源", kind, version)
	}

	// 本地 zip 安装 (自定义版本可能直接指定本地文件, 无需下载)
	if entry.LocalZip != "" && len(entry.AllURLs()) == 0 {
		return a.installFromZip(kind, version, entry.LocalZip, entry.RootInZip)
	}

	urls := entry.AllURLs()
	if len(urls) == 0 {
		return fmt.Errorf("%s %s 没有配置任何下载 URL", kind, version)
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

	// 安装后必须验证关键二进制存在 (符合"什么样的 zip 算合格"的条件)
	if missing := sources.VerifyInstall(kind, dest); len(missing) > 0 {
		msg := fmt.Sprintf("%s 安装后验证失败, 缺少关键文件: %v\n请确认 rootInZip 设置正确, 或 zip 文件结构与官方版本一致", kind, missing)
		wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "error": msg})
		return fmt.Errorf(msg)
	}

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
	case "redis":
		st.RedisVersion = version
	}
	_ = state.Save(st)

	logger.Info("%s %s 安装完成", kind, version)
	wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "success": true})
	return nil
}

// ============ 自定义版本 ============

// ExpectedBinaries 返回该组件安装后需要存在的关键文件 (相对安装目录).
// 前端可在自定义版本对话框里展示给用户看, 让他知道 zip 必须含哪些文件.
func (a *App) ExpectedBinaries(kind string) []string {
	return sources.ExpectedBinaries(kind)
}

// AddCustomVersion 添加用户自定义版本 (URL 模式).
// 写入 config/custom_sources.json. 同 version 会覆盖.
func (a *App) AddCustomVersion(kind, version string, urls []string, rootInZip string) error {
	if version == "" {
		return fmt.Errorf("版本号不能为空")
	}
	if !isValidKind(kind) {
		return fmt.Errorf("不支持的组件: %s", kind)
	}
	if len(urls) == 0 {
		return fmt.Errorf("至少需要一个下载 URL")
	}
	cleanUrls := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return fmt.Errorf("URL 必须以 http:// 或 https:// 开头: %s", u)
		}
		cleanUrls = append(cleanUrls, u)
	}
	if len(cleanUrls) == 0 {
		return fmt.Errorf("没有有效的 URL")
	}

	custom, _ := sources.LoadCustom()
	if custom == nil {
		custom = &sources.Custom{}
	}
	custom.Upsert(kind, sources.VersionEntry{
		Version:   version,
		URLs:      cleanUrls,
		RootInZip: rootInZip,
	})
	if err := sources.SaveCustom(custom); err != nil {
		return err
	}
	logger.Info("自定义 %s 版本已添加: %s (%d 个 URL)", kind, version, len(cleanUrls))
	return nil
}

// AddCustomVersionLocal 添加用户自定义版本 (本地 zip 模式).
// 不立即安装, 只把记录保存. 在版本列表里能选到, 选了之后点"开始安装"才执行.
func (a *App) AddCustomVersionLocal(kind, version, zipPath, rootInZip string) error {
	if version == "" {
		return fmt.Errorf("版本号不能为空")
	}
	if !isValidKind(kind) {
		return fmt.Errorf("不支持的组件: %s", kind)
	}
	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("本地文件不存在: %s", zipPath)
	}
	if !strings.HasSuffix(strings.ToLower(zipPath), ".zip") {
		return fmt.Errorf("文件必须是 .zip")
	}

	custom, _ := sources.LoadCustom()
	if custom == nil {
		custom = &sources.Custom{}
	}
	custom.Upsert(kind, sources.VersionEntry{
		Version:   version,
		LocalZip:  zipPath,
		RootInZip: rootInZip,
	})
	if err := sources.SaveCustom(custom); err != nil {
		return err
	}
	logger.Info("自定义 %s 版本已添加 (本地 zip): %s", kind, version)
	return nil
}

// RemoveCustomVersion 删除一个用户自定义版本 (不影响内置版本).
func (a *App) RemoveCustomVersion(kind, version string) error {
	custom, _ := sources.LoadCustom()
	if custom == nil {
		return nil
	}
	if !custom.Remove(kind, version) {
		return fmt.Errorf("未找到自定义版本: %s %s", kind, version)
	}
	if err := sources.SaveCustom(custom); err != nil {
		return err
	}
	logger.Info("已删除自定义 %s 版本: %s", kind, version)
	return nil
}

// installFromZip 内部: 从本地 zip 安装 (供 InstallComponent 在 LocalZip 时调用).
func (a *App) installFromZip(kind, version, zipPath, rootInZip string) error {
	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("本地文件不存在: %s", zipPath)
	}
	wruntime.EventsEmit(a.ctx, "install:start", map[string]any{"kind": kind, "version": version})
	logger.Info("从本地 zip 安装 %s %s: %s", kind, version, zipPath)

	dest := destDir(kind)
	a.stopFor(kind)
	if err := extract.Zip(zipPath, dest, rootInZip); err != nil {
		wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "error": err.Error()})
		return err
	}

	if missing := sources.VerifyInstall(kind, dest); len(missing) > 0 {
		msg := fmt.Sprintf("%s 安装后验证失败, 缺少关键文件: %v\n请确认 rootInZip 设置正确 (zip 内的子目录名), 或换一个 zip", kind, missing)
		wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "error": msg})
		return fmt.Errorf(msg)
	}

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
	case "redis":
		st.RedisVersion = version
	}
	_ = state.Save(st)

	logger.Info("%s %s (本地 zip) 安装完成", kind, version)
	wruntime.EventsEmit(a.ctx, "install:done", map[string]any{"kind": kind, "version": version, "success": true})
	return nil
}

// PickLocalZip 打开文件选择器, 让用户选本地 zip 文件.
func (a *App) PickLocalZip() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择本地 zip 文件",
		Filters: []wruntime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
}

func isValidKind(kind string) bool {
	switch kind {
	case "nginx", "php", "mysql", "postgresql", "postgres", "redis":
		return true
	}
	return false
}

// TestUrl 探测单个 URL 是否可达 (HEAD / Range 兜底), 不下载内容.
func (a *App) TestUrl(url string) nettest.Result {
	return nettest.Test(a.ctx, url)
}

// TestUrls 并发探测多个 URL (前端 "测试连通性" 按钮调它).
func (a *App) TestUrls(urls []string) []nettest.Result {
	return nettest.TestMany(a.ctx, urls)
}

// PreviewUrls 返回该版本下载会按序尝试的 URL 列表 (供前端展示).
func (a *App) PreviewUrls(kind, version string) ([]string, error) {
	src, err := sources.Load()
	if err != nil {
		return nil, err
	}
	entry := src.Find(kind, version)
	if entry == nil {
		return nil, fmt.Errorf("未找到 %s %s", kind, version)
	}
	return entry.AllURLs(), nil
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
	case "redis":
		_ = autostart.UnregisterService(services.RedisServiceName)
		_ = a.redis.Stop()
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
	case "redis":
		st.RedisVersion = ""
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
	case "redis":
		return paths.RedisDir
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
	case "redis":
		_ = a.redis.Stop()
	}
}

// initConfigFor 调对应组件的 InitConfig, 把错误记到日志而不是吞掉.
// 这样如果配置文件没写成功, 用户能在日志里看到原因.
func (a *App) initConfigFor(kind string) {
	var err error
	switch kind {
	case "nginx":
		err = a.nginx.InitConfig()
	case "php":
		err = a.php.InitConfig()
	case "mysql":
		err = a.mysql.InitConfig()
	case "postgresql", "postgres":
		err = a.pg.InitConfig()
	case "redis":
		err = a.redis.InitConfig()
	}
	if err != nil {
		logger.Error("%s 配置初始化失败: %v", kind, err)
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
	case "redis":
		return filepath.Join(paths.RedisDir, "redis.windows.conf")
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

// MysqlCreateUserInput 创建 MySQL 用户/库 的入参.
type MysqlCreateUserInput struct {
	RootPwd string `json:"rootPwd"` // root 密码 (无密码就传空串)
	DbName  string `json:"dbName"`  // 想同时创建/授权的库名 (空就只建用户)
	User    string `json:"user"`    // 新用户名
	UserPwd string `json:"userPwd"` // 新用户密码
	Host    string `json:"host"`    // 允许连接的主机 (默认 localhost, % 表示任意)
}

// MysqlCreateUser 创建用户 + 可选建库 + GRANT ALL.
func (a *App) MysqlCreateUser(in MysqlCreateUserInput) error {
	return a.mysql.CreateUser(in.RootPwd, in.DbName, in.User, in.UserPwd, in.Host)
}

// MysqlListDatabases 列出所有数据库 (含字符集 + 系统库标记).
func (a *App) MysqlListDatabases(rootPwd string) ([]services.MysqlDatabase, error) {
	return a.mysql.ListDatabases(rootPwd)
}

// MysqlListUsers 列出所有 MySQL 账号 (User@Host).
func (a *App) MysqlListUsers(rootPwd string) ([]services.MysqlUser, error) {
	return a.mysql.ListUsers(rootPwd)
}

// MysqlDropDatabase 删库 (系统库禁删).
func (a *App) MysqlDropDatabase(name, rootPwd string) error {
	if err := a.mysql.DropDatabase(name, rootPwd); err != nil {
		return err
	}
	logger.Info("MySQL 数据库已删除: %s", name)
	return nil
}

// MysqlDropUser 删账号 (root / 系统账号禁删).
func (a *App) MysqlDropUser(user, host, rootPwd string) error {
	if err := a.mysql.DropUser(user, host, rootPwd); err != nil {
		return err
	}
	logger.Info("MySQL 账号已删除: %s@%s", user, host)
	return nil
}

// RedisGetPassword 返回 redis.windows.conf 里当前的 requirepass (空串 = 无密码).
func (a *App) RedisGetPassword() string { return a.redis.Password() }

// RedisSetPassword 改 Redis 密码. 空串表示移除. 运行中会立即生效.
func (a *App) RedisSetPassword(newPwd string) error { return a.redis.SetPassword(newPwd) }

// ============ PHP 扩展 ============

func (a *App) PhpExtensions() []services.Extension       { return a.php.ListExtensions() }
func (a *App) PhpSetExtension(name string, enable bool) error {
	if err := a.php.SetExtension(name, enable); err != nil {
		return err
	}
	logger.Info("PHP 扩展 %s -> %v (重启 PHP-CGI 生效)", name, enable)
	return nil
}

// ============ PHP 防跨盘访问 (open_basedir) ============

// OpenBasedirInfo 给前端展示当前配置 + 生效后的完整路径列表.
type OpenBasedirInfo struct {
	Enabled        bool     `json:"enabled"`        // 是否启用 open_basedir 限制
	Extra          []string `json:"extra"`          // 用户配置的额外允许目录
	EffectivePaths []string `json:"effectivePaths"` // 实际会写入 ini 的完整列表
	AlwaysPaths    []string `json:"alwaysPaths"`    // 始终允许 (www + php + tmp) 不可改
}

// GetOpenBasedir 返回 PHP 防跨盘访问当前配置.
func (a *App) GetOpenBasedir() OpenBasedirInfo {
	st := state.Load()
	always := []string{
		filepath.ToSlash(paths.WwwDir) + "/",
		filepath.ToSlash(paths.PhpDir) + "/",
		filepath.ToSlash(paths.TmpDir) + "/",
	}
	return OpenBasedirInfo{
		Enabled:        st.OpenBasedir,
		Extra:          st.OpenBasedirExtra,
		EffectivePaths: a.php.OpenBasedirPaths(),
		AlwaysPaths:    always,
	}
}

// SetOpenBasedir 写入 state + 同步到 php.ini. 需要重启 PHP-CGI 才生效.
// extra 里每行一个路径 (例如 "D:\\", "E:\\data\\"). 空行会被清掉.
func (a *App) SetOpenBasedir(enabled bool, extra []string) error {
	clean := make([]string, 0, len(extra))
	seen := map[string]bool{}
	for _, e := range extra {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		key := strings.ToLower(filepath.ToSlash(e))
		if seen[key] {
			continue
		}
		seen[key] = true
		clean = append(clean, e)
	}
	st := state.Load()
	st.OpenBasedir = enabled
	st.OpenBasedirExtra = clean
	if err := state.Save(st); err != nil {
		return err
	}
	if _, err := os.Stat(a.php.IniPath()); err != nil {
		// PHP 没装就只存 state, 装完后 InitConfig 会自动应用
		logger.Info("PHP 未安装, 防跨盘配置已保存 state, 装完 PHP 自动生效")
		return nil
	}
	if err := a.php.ApplyOpenBasedir(); err != nil {
		return err
	}
	logger.Info("PHP 防跨盘 open_basedir -> %v (重启 PHP-CGI 生效)", enabled)
	return nil
}

// ListNonSystemDrives 枚举本机所有非 C 盘的可用盘符 (D:\ E:\ ...).
// 前端 "一键添加非 C 盘" 按钮调用, 把这些盘加进 OpenBasedirExtra.
func (a *App) ListNonSystemDrives() []string {
	var drives []string
	for c := 'D'; c <= 'Z'; c++ {
		p := string(c) + ":\\"
		if _, err := os.Stat(p); err == nil {
			drives = append(drives, p)
		}
	}
	return drives
}

// PhpInstallableExts 返回可在线安装的 PHP 扩展清单 (内置常用: redis, memcached, mongodb, xdebug ...).
func (a *App) PhpInstallableExts() []services.InstallableExt {
	return services.KnownInstallableExts()
}

// PhpInstallExtension 从 PECL 在线下载并安装一个 PHP 扩展.
// 自动检测 PHP 版本和 VS 编译标签, 构造 PECL Windows zip URL,
// 下载 → 解压 → 拷贝 *.dll 到 ext/ → 改 php.ini 启用. 重启 PHP-CGI 生效.
func (a *App) PhpInstallExtension(name, extVer string) error {
	ctx := a.registerCancel("phpext:" + name)
	defer a.clearCancel("phpext:" + name)
	prog := func(d, t int64) {
		wruntime.EventsEmit(a.ctx, "phpext:progress", map[string]any{
			"name": name, "version": extVer, "loaded": d, "total": t,
		})
	}
	return a.php.InstallExtensionFromPECL(ctx, name, extVer, prog)
}

// ============ 自启 ============

func (a *App) EnsureNssm() error {
	prog := func(d, t int64) {
		wruntime.EventsEmit(a.ctx, "nssm:progress", map[string]any{"loaded": d, "total": t})
	}
	ctx := a.registerCancel("nssm")
	defer a.clearCancel("nssm")
	_, err := autostart.EnsureNssm(ctx, prog)
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
		mk("redis", "Redis 服务", services.RedisServiceName, a.redis.Status()),
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
	if err := a.EnsureNssm(); err != nil {
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
	case "redis":
		args := []string{}
		if _, err := os.Stat(a.redis.ConfPath()); err == nil {
			args = []string{a.redis.ConfPath()}
		}
		return autostart.RegisterService(services.RedisServiceName,
			a.redis.ExePath(), args,
			paths.RedisDir, "WinPHP Redis", nil)
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
	case "redis":
		return autostart.UnregisterService(services.RedisServiceName)
	}
	return fmt.Errorf("未知: %s", key)
}

func (a *App) EnableAllAutoStart() error {
	keys := []string{"panel", "nginx", "php", "mysql", "postgres", "redis"}
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
	for _, k := range []string{"panel", "nginx", "php", "mysql", "postgres", "redis"} {
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
	case "redis":
		return a.redis.Status()
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

// ============ 主题 ============

// ListThemes 列出所有可用主题 (内置 + themes/ 目录里的自定义).
func (a *App) ListThemes() []theme.Info {
	return theme.List()
}

// GetCurrentTheme 返回当前应用的主题 (含 CSS 给前端注入).
// 启动时调一次, 应用持久化的主题.
func (a *App) GetCurrentTheme() theme.Applied {
	st := state.Load()
	id := st.Theme
	if id == "" {
		id = "default"
	}
	applied, err := theme.Get(id)
	if err != nil {
		// 主题被删了, fallback default
		applied, _ = theme.Get("default")
	}
	return applied
}

// SetTheme 切换主题. 持久化到 state.
func (a *App) SetTheme(id string) (theme.Applied, error) {
	if !theme.ValidID(id) {
		return theme.Applied{}, fmt.Errorf("非法主题 id: %s", id)
	}
	applied, err := theme.Get(id)
	if err != nil {
		return theme.Applied{}, err
	}
	st := state.Load()
	st.Theme = id
	_ = state.Save(st)
	logger.Info("切换主题: %s (%s)", applied.Info.Name, id)
	return applied, nil
}

// RemoveCustomTheme 删除一个自定义主题目录 (不能删内置).
func (a *App) RemoveCustomTheme(id string) error {
	return theme.RemoveCustom(id)
}

// OpenThemesFolder 打开 themes/ 目录给开发者放主题包.
func (a *App) OpenThemesFolder() error {
	return execOpen(paths.ThemesDir)
}

// ============ 壁纸 (二次元美化) ============

// Wallpaper 自定义壁纸返回结构.
type Wallpaper struct {
	DataURL string `json:"dataUrl"` // base64 data URL, 前端直接 set 到 background-image
	Path    string `json:"path"`    // 本地保存路径
	Empty   bool   `json:"empty"`   // 没壁纸时为 true
}

var wallpaperExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp"}

func base64Encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func wallpaperMime(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	}
	return "application/octet-stream"
}

// GetWallpaper 加载当前已设置的壁纸 (启动时前端会调).
func (a *App) GetWallpaper() Wallpaper {
	for _, ext := range wallpaperExts {
		p := filepath.Join(paths.ConfigDir, "wallpaper"+ext)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		return Wallpaper{
			DataURL: "data:" + wallpaperMime(ext) + ";base64," + base64Encode(data),
			Path:    p,
		}
	}
	return Wallpaper{Empty: true}
}

// PickAndSetWallpaper 弹文件对话框让用户选图, 复制到 config/ 并返回 dataURL.
func (a *App) PickAndSetWallpaper() (Wallpaper, error) {
	selected, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择壁纸图片",
		Filters: []wruntime.FileFilter{
			{DisplayName: "图片 (*.jpg;*.jpeg;*.png;*.webp;*.gif;*.bmp)",
				Pattern: "*.jpg;*.jpeg;*.png;*.webp;*.gif;*.bmp"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return Wallpaper{Empty: true}, err
	}
	if selected == "" {
		return Wallpaper{Empty: true}, nil
	}
	return a.setWallpaperFromFile(selected)
}

func (a *App) setWallpaperFromFile(src string) (Wallpaper, error) {
	ext := strings.ToLower(filepath.Ext(src))
	valid := false
	for _, e := range wallpaperExts {
		if e == ext {
			valid = true
			break
		}
	}
	if !valid {
		return Wallpaper{Empty: true}, fmt.Errorf("不支持的图片格式: %s (仅 jpg/png/webp/gif/bmp)", ext)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return Wallpaper{Empty: true}, err
	}
	// 大小限制 (10 MB)
	if len(data) > 10*1024*1024 {
		return Wallpaper{Empty: true}, fmt.Errorf("壁纸文件过大 (%.1f MB), 建议 < 10 MB", float64(len(data))/1024/1024)
	}
	target := filepath.Join(paths.ConfigDir, "wallpaper"+ext)
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return Wallpaper{Empty: true}, err
	}
	// 删除其他扩展名旧壁纸 (确保只有一个)
	for _, oldExt := range wallpaperExts {
		if oldExt == ext {
			continue
		}
		_ = os.Remove(filepath.Join(paths.ConfigDir, "wallpaper"+oldExt))
	}
	logger.Info("壁纸已更新: %s (%.1f KB)", target, float64(len(data))/1024)
	return Wallpaper{
		DataURL: "data:" + wallpaperMime(ext) + ";base64," + base64Encode(data),
		Path:    target,
	}, nil
}

// ClearWallpaper 删除当前壁纸.
func (a *App) ClearWallpaper() error {
	for _, ext := range wallpaperExts {
		_ = os.Remove(filepath.Join(paths.ConfigDir, "wallpaper"+ext))
	}
	logger.Info("壁纸已清除")
	return nil
}

// ============ 数据目录 (升级保留数据) ============

type DataDirInfo struct {
	Current      string `json:"current"`      // 当前实际生效的数据目录
	ExeDir       string `json:"exeDir"`       // EXE 所在目录 (指针文件总是放这里)
	PointerExist bool   `json:"pointerExist"` // EXE 旁是否已有 data-dir.txt
	PointerPath  string `json:"pointerPath"`  // 指针文件的完整路径
}

// GetDataDirInfo 返回当前数据目录及指针状态.
func (a *App) GetDataDirInfo() DataDirInfo {
	info := DataDirInfo{
		Current:     paths.Root,
		ExeDir:      paths.ExeDir,
		PointerPath: paths.PointerPath(),
	}
	if _, err := os.Stat(info.PointerPath); err == nil {
		info.PointerExist = true
	}
	return info
}

// PickDirectory 通用文件夹选择对话框. 前端"浏览..."按钮调用.
// title 标题 (空就用默认), defaultDir 起始目录 (空就用 www 目录).
// 返回用户选中的绝对路径; 取消返回空串.
func (a *App) PickDirectory(title, defaultDir string) (string, error) {
	if title == "" {
		title = "选择文件夹"
	}
	if defaultDir == "" {
		defaultDir = paths.WwwDir
	}
	selected, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:                title,
		DefaultDirectory:     defaultDir,
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", err
	}
	return selected, nil
}

// PickAndSetDataDir 弹文件夹选择对话框, 把目标路径写入 EXE 旁的 data-dir.txt.
// 用户需重启面板才生效 (返回前不切换 paths.Root, 避免运行中乱).
func (a *App) PickAndSetDataDir() (DataDirInfo, error) {
	selected, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:                "选择新的数据目录 (bin / www / config / logs 都将放这里)",
		DefaultDirectory:     paths.Root,
		CanCreateDirectories: true,
	})
	if err != nil {
		return a.GetDataDirInfo(), err
	}
	if selected == "" {
		return a.GetDataDirInfo(), nil
	}
	if err := paths.SetDataDirPointer(selected); err != nil {
		return a.GetDataDirInfo(), err
	}
	logger.Info("数据目录指针已写入 (重启面板生效): %s -> %s", paths.PointerPath(), selected)
	return a.GetDataDirInfo(), nil
}

// ResetDataDir 删除指针文件, 回到默认 (数据放 EXE 同目录).
func (a *App) ResetDataDir() (DataDirInfo, error) {
	if err := paths.SetDataDirPointer(""); err != nil {
		return a.GetDataDirInfo(), err
	}
	logger.Info("数据目录指针已删除, 重启后数据回到 EXE 目录")
	return a.GetDataDirInfo(), nil
}

// ============ 端口检测 ============

func (a *App) PortInUse(port int) bool { return proc.PortListening(port) }

// DiagnosePort 返回端口的完整诊断信息: 是否占用, 哪个进程占用 (PID/名字),
// 是否被 Windows 系统预留 (HNS/WSL/Docker 常见), 以及友好建议字符串.
func (a *App) DiagnosePort(port int) portcheck.PortInfo {
	return portcheck.Diagnose(port)
}
