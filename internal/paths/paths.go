package paths

import (
	"os"
	"path/filepath"
)

// 全局路径. 所有组件二进制、网站、配置、日志都放在面板可执行文件所在目录下.

var (
	Root      string // 面板根目录
	BinDir    string // bin/
	NginxDir  string // bin/nginx
	PhpDir    string // bin/php
	MysqlDir  string // bin/mysql
	PgDir     string // bin/postgresql
	RedisDir  string // bin/redis
	WwwDir    string // www
	LogsDir   string // logs
	TmpDir    string // tmp
	ConfigDir string // config
	TplDir    string // config/templates
	ThemesDir string // themes/  (第三方主题包)

	SitesFile  string // config/sites.json
	StateFile  string // config/state.json
	SourceFile string // config/sources.json
	NssmFile   string // bin/nssm.exe

	HostsFile string // C:\Windows\System32\drivers\etc\hosts
)

func Init() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	Root = filepath.Dir(exe)

	// 开发模式: 当 go run / wails dev 时, exe 在 tmp 目录, 用 cwd
	if env := os.Getenv("WINPHP_DEV_ROOT"); env != "" {
		Root = env
	}

	BinDir = filepath.Join(Root, "bin")
	NginxDir = filepath.Join(BinDir, "nginx")
	PhpDir = filepath.Join(BinDir, "php")
	MysqlDir = filepath.Join(BinDir, "mysql")
	PgDir = filepath.Join(BinDir, "postgresql")
	RedisDir = filepath.Join(BinDir, "redis")
	WwwDir = filepath.Join(Root, "www")
	LogsDir = filepath.Join(Root, "logs")
	TmpDir = filepath.Join(Root, "tmp")
	ConfigDir = filepath.Join(Root, "config")
	TplDir = filepath.Join(ConfigDir, "templates")
	ThemesDir = filepath.Join(Root, "themes")

	SitesFile = filepath.Join(ConfigDir, "sites.json")
	StateFile = filepath.Join(ConfigDir, "state.json")
	SourceFile = filepath.Join(ConfigDir, "sources.json")
	NssmFile = filepath.Join(BinDir, "nssm.exe")

	HostsFile = filepath.Join(os.Getenv("SystemRoot"), "System32", "drivers", "etc", "hosts")
	if os.Getenv("SystemRoot") == "" {
		HostsFile = `C:\Windows\System32\drivers\etc\hosts`
	}

	// 初始化目录
	dirs := []string{BinDir, WwwDir, LogsDir, TmpDir, ConfigDir, TplDir, ThemesDir, filepath.Join(WwwDir, "default")}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
