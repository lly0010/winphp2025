package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/portcheck"
	"github.com/lly0010/winphp2025/internal/proc"
	"github.com/lly0010/winphp2025/internal/state"
)

const PhpServiceName = "WinPHPPhp"

type PHP struct{}

func (PHP) Name() string     { return "php" }
func (PHP) ExePath() string  { return filepath.Join(paths.PhpDir, "php.exe") }
func (PHP) CgiPath() string  { return filepath.Join(paths.PhpDir, "php-cgi.exe") }
func (PHP) IniPath() string  { return filepath.Join(paths.PhpDir, "php.ini") }

func (p PHP) Version() string {
	if _, err := os.Stat(p.ExePath()); err != nil {
		return ""
	}
	out, err := runHidden(p.ExePath(), 3*time.Second, "-v")
	if err == nil {
		// First line: "PHP 8.3.14 (cli) ..."
		if i := strings.Index(out, "PHP "); i >= 0 {
			s := out[i+4:]
			end := strings.IndexAny(s, " \r\n\t")
			if end > 0 {
				return s[:end]
			}
		}
	}
	// php.exe 跑不起来 (常见: 缺 VC++ 2022 Redistributable),
	// 或者输出格式异常. 回退到安装时写入 state 的版本号.
	if st := state.Load(); st.PhpVersion != "" {
		return st.PhpVersion
	}
	return ""
}

func (p PHP) Status() Status {
	binOk := false
	if _, err := os.Stat(p.CgiPath()); err == nil {
		binOk = true
	}
	svcInstalled := ServiceExists(PhpServiceName)
	svc := ""
	if svcInstalled {
		svc = ServiceStatusStr(PhpServiceName)
	}
	running := svc == "Running"
	if !running && binOk {
		running = proc.HasProcessByPathPrefix("php-cgi", paths.PhpDir)
	}
	if !running && binOk {
		running = proc.PortListening(9000)
	}
	return Status{
		Running: running, Port: 9000, Version: p.Version(),
		ServiceInstalled: svcInstalled, ServiceStatus: svc, BinInstalled: binOk,
	}
}

func (p PHP) Start() error {
	exe := p.CgiPath()
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("PHP 未安装")
	}
	if p.Status().Running {
		return fmt.Errorf("PHP-CGI 已运行")
	}
	// 自我修复 php.ini
	if _, err := os.Stat(p.IniPath()); err != nil {
		logger.Warn("php.ini 不存在, 自动重新生成")
		if e := (PHP{}).InitConfig(); e != nil {
			return fmt.Errorf("php.ini 不存在, 自动生成失败: %v", e)
		}
	}
	if proc.PortListening(9000) {
		return fmt.Errorf("端口 9000 已被占用. %s", portcheck.Diagnose(9000).Diagnosis)
	}
	if ServiceExists(PhpServiceName) {
		return StartService(PhpServiceName)
	}
	cmd := exec.Command(exe, "-b", "127.0.0.1:9000", "-c", p.IniPath())
	cmd.Dir = paths.PhpDir
	cmd.Env = append(os.Environ(),
		"PHP_FCGI_CHILDREN=5",
		"PHP_FCGI_MAX_REQUESTS=1000",
	)
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	time.Sleep(700 * time.Millisecond)
	logger.Info("PHP-CGI 启动")
	return nil
}

func (p PHP) Stop() error {
	if ServiceExists(PhpServiceName) {
		_ = StopService(PhpServiceName)
	}
	killByPathPrefix("php-cgi", paths.PhpDir)
	time.Sleep(200 * time.Millisecond)
	logger.Info("PHP-CGI 已停止")
	return nil
}

func (p PHP) Restart() error {
	_ = p.Stop()
	time.Sleep(300 * time.Millisecond)
	return p.Start()
}

const defaultPhpIni = `[PHP]
engine = On
short_open_tag = Off
expose_php = Off
max_execution_time = 300
memory_limit = 256M
error_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT
display_errors = On
log_errors = On
error_log = "##PHP_DIR##/logs/php_error.log"
post_max_size = 64M
file_uploads = On
upload_max_filesize = 64M
default_charset = "UTF-8"
extension_dir = "ext"
enable_dl = Off
cgi.force_redirect = 0
cgi.fix_pathinfo = 1
fastcgi.impersonate = 1
default_socket_timeout = 60

[Date]
date.timezone = "Asia/Shanghai"

[mbstring]
mbstring.internal_encoding = UTF-8

extension=bz2
extension=curl
extension=fileinfo
extension=gd
extension=gettext
extension=mbstring
extension=exif
extension=mysqli
extension=openssl
extension=pdo_mysql
extension=pdo_pgsql
extension=pdo_sqlite
extension=pgsql
extension=sqlite3
extension=intl
extension=soap
extension=sockets
extension=xsl
extension=zip

[opcache]
zend_extension=opcache
opcache.enable=1
opcache.enable_cli=0
opcache.memory_consumption=128
opcache.interned_strings_buffer=16
opcache.max_accelerated_files=10000
opcache.revalidate_freq=2

[Session]
session.save_handler = files
session.save_path = "##PHP_DIR##/tmp"
session.cookie_httponly = 1
session.cookie_samesite = "Lax"
`

func (PHP) InitConfig() error {
	if _, err := os.Stat(paths.PhpDir); err != nil {
		return err
	}
	tpl, _ := readTemplate("php.ini", defaultPhpIni)
	conf := strings.ReplaceAll(tpl, "##PHP_DIR##", filepath.ToSlash(paths.PhpDir))
	if err := os.WriteFile(filepath.Join(paths.PhpDir, "php.ini"), []byte(conf), 0o644); err != nil {
		return err
	}
	for _, d := range []string{"logs", "tmp"} {
		_ = os.MkdirAll(filepath.Join(paths.PhpDir, d), 0o755)
	}
	logger.Info("PHP 配置初始化完成")
	return nil
}

// ListExtensions 列出 ext/ 下所有 DLL, 并标记当前是否启用 (用于前端扩展管理)
type Extension struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // "extension" or "zend_extension"
}

func (p PHP) ListExtensions() []Extension {
	extDir := filepath.Join(paths.PhpDir, "ext")
	entries, err := os.ReadDir(extDir)
	if err != nil {
		return nil
	}
	// 读 php.ini 找出当前启用的扩展
	iniText, _ := readFileAll(p.IniPath())
	enabled := map[string]string{}
	for _, line := range strings.Split(iniText, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, ";") || l == "" {
			continue
		}
		if strings.HasPrefix(l, "extension=") {
			name := strings.TrimPrefix(l, "extension=")
			name = strings.Trim(name, "\"' \t\r")
			enabled[strings.ToLower(name)] = "extension"
		}
		if strings.HasPrefix(l, "zend_extension=") {
			name := strings.TrimPrefix(l, "zend_extension=")
			name = strings.Trim(name, "\"' \t\r")
			enabled[strings.ToLower(name)] = "zend_extension"
		}
	}
	var exts []Extension
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasPrefix(strings.ToLower(n), "php_") || !strings.HasSuffix(strings.ToLower(n), ".dll") {
			continue
		}
		short := strings.TrimPrefix(strings.ToLower(n), "php_")
		short = strings.TrimSuffix(short, ".dll")
		typ, ok := enabled[short]
		if !ok {
			typ = "extension"
		}
		exts = append(exts, Extension{Name: short, Enabled: ok, Type: typ})
	}
	return exts
}

// SetExtension 启用/禁用扩展. 修改 php.ini 内的 ;extension=xxx 行.
func (p PHP) SetExtension(name string, enabled bool) error {
	iniPath := p.IniPath()
	text, err := readFileAll(iniPath)
	if err != nil {
		return err
	}
	lines := strings.Split(text, "\n")
	// 先尝试切换已有行
	matched := false
	for i, line := range lines {
		l := strings.TrimSpace(line)
		isExt := strings.HasPrefix(strings.TrimLeft(l, ";"), "extension=")
		if !isExt {
			continue
		}
		// 提取扩展名
		body := strings.TrimLeft(l, "; \t")
		val := strings.TrimPrefix(body, "extension=")
		val = strings.Trim(val, "\"' \t\r")
		if strings.EqualFold(val, name) {
			if enabled {
				lines[i] = "extension=" + name
			} else {
				lines[i] = ";extension=" + name
			}
			matched = true
		}
	}
	if !matched && enabled {
		// 不存在则追加到末尾
		lines = append(lines, "extension="+name)
	}
	out := strings.Join(lines, "\n")
	return os.WriteFile(iniPath, []byte(out), 0o644)
}
