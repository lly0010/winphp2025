package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// 全局路径. 所有组件二进制、网站、配置、日志都放在 Root 目录下.
// Root 的选择优先级:
//   1. 环境变量 WINPHP_DATA_DIR (开发用)
//   2. EXE 旁的 data-dir.txt 里写的路径 (用户在工具页设置后写入)
//   3. EXE 所在目录 (默认, 向后兼容)
// 用 data-dir.txt 把数据目录跟 EXE 分开, 更新 EXE 时数据完全独立不丢.

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

// PointerFileName 在 EXE 旁的指针文件名, 写一行路径就把 Root 切到那里.
const PointerFileName = "data-dir.txt"

// ExeDir 返回 EXE 所在目录 (无论 Root 被切到哪, 这个总是固定的).
// 指针文件 data-dir.txt 永远写在这里.
var ExeDir string

func Init() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	ExeDir = filepath.Dir(exe)
	Root = ExeDir

	// 1) 环境变量 (最高优先级, 开发用)
	if env := os.Getenv("WINPHP_DEV_ROOT"); env != "" {
		Root = env
	} else if env := os.Getenv("WINPHP_DATA_DIR"); env != "" {
		Root = env
	} else {
		// 2) 指针文件
		if b, err := os.ReadFile(filepath.Join(ExeDir, PointerFileName)); err == nil {
			if p := strings.TrimSpace(string(b)); p != "" {
				if abs, err := filepath.Abs(p); err == nil {
					Root = abs
				}
			}
		}
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

// PointerPath 返回指针文件的绝对路径 (永远在 EXE 旁, 跟数据目录无关).
func PointerPath() string {
	return filepath.Join(ExeDir, PointerFileName)
}

// SetDataDirPointer 把目标路径写入指针文件. 调用后需重启面板才生效.
// 传空字符串则删除指针 (回到默认: 数据放 EXE 同目录).
func SetDataDirPointer(target string) error {
	p := PointerPath()
	if target == "" {
		err := os.Remove(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(abs+"\n"), 0o644)
}
