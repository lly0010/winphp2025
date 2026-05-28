package state

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/lly0010/winphp2025/internal/paths"
)

type State struct {
	NginxVersion string `json:"nginxVersion"`
	PhpVersion   string `json:"phpVersion"`
	MysqlVersion string `json:"mysqlVersion"`
	MysqlInited  bool   `json:"mysqlInited"`
	PgVersion    string `json:"pgVersion"`
	PgInited     bool   `json:"pgInited"`
	RedisVersion string `json:"redisVersion"`
	Theme        string `json:"theme,omitempty"` // 当前主题 id (default / blue-classic / custom:xxx)
}

type Site struct {
	Name       string `json:"name"`
	ServerName string `json:"serverName"`
	Root       string `json:"root"`
	Port       int    `json:"port"`
	PhpVersion string `json:"phpVersion,omitempty"`
	Template   string `json:"template,omitempty"`
	Rewrite    string `json:"rewrite,omitempty"` // 伪静态: default / thinkphp / discuz / none
	CreatedAt  string `json:"createdAt"`
}

var mu sync.Mutex

func Load() State {
	mu.Lock()
	defer mu.Unlock()
	var s State
	b, err := os.ReadFile(paths.StateFile)
	if err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func Save(s State) error {
	mu.Lock()
	defer mu.Unlock()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.StateFile, b, 0o644)
}

func Sites() []Site {
	mu.Lock()
	defer mu.Unlock()
	var ss []Site
	b, err := os.ReadFile(paths.SitesFile)
	if err == nil {
		_ = json.Unmarshal(b, &ss)
	}
	if ss == nil {
		ss = []Site{}
	}
	return ss
}

func SaveSites(ss []Site) error {
	mu.Lock()
	defer mu.Unlock()
	if ss == nil {
		ss = []Site{}
	}
	b, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.SitesFile, b, 0o644)
}
