package sources

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lly0010/winphp2025/internal/paths"
)

// 内嵌默认的 sources.json. 用户也可以通过编辑 config/sources.json 覆盖.

//go:embed sources_default.json
var defaultJSON []byte

type Sources struct {
	Comment    string                  `json:"_comment,omitempty"`
	Nssm       NssmEntry               `json:"nssm"`
	Nginx      []VersionEntry          `json:"nginx"`
	Php        []VersionEntry          `json:"php"`
	Mysql      []VersionEntry          `json:"mysql"`
	Postgresql []VersionEntry          `json:"postgresql"`
	Redis      []VersionEntry          `json:"redis"`
	raw        map[string]interface{} `json:"-"`
}

type NssmEntry struct {
	Version  string   `json:"version"`
	URL      string   `json:"url"`
	URLs     []string `json:"urls,omitempty"`
	ExeInZip string   `json:"exeInZip"`
}

type VersionEntry struct {
	Version   string   `json:"version"`
	URLs      []string `json:"urls"`
	URL       string   `json:"url,omitempty"`      // 兼容旧字段
	RootInZip string   `json:"rootInZip"`
	VsTag     string   `json:"vs,omitempty"`       // PHP: vs16 / vs17 / vc15
	Custom    bool     `json:"custom,omitempty"`   // 是否用户自定义版本
	LocalZip  string   `json:"localZip,omitempty"` // 本地 zip 文件路径 (没有 URL 时使用)
}

// AllURLs 返回该版本所有下载 URL (按顺序尝试).
func (e *VersionEntry) AllURLs() []string {
	if len(e.URLs) > 0 {
		return append([]string(nil), e.URLs...)
	}
	if e.URL != "" {
		return []string{e.URL}
	}
	return nil
}

// AllURLs for NSSM
func (n *NssmEntry) AllURLs() []string {
	if len(n.URLs) > 0 {
		return append([]string(nil), n.URLs...)
	}
	if n.URL != "" {
		return []string{n.URL}
	}
	return nil
}

// Load 优先读 config/sources.json, 不存在则用内嵌默认.
func Load() (*Sources, error) {
	var raw []byte
	if _, err := os.Stat(paths.SourceFile); err == nil {
		raw, err = os.ReadFile(paths.SourceFile)
		if err != nil {
			return nil, fmt.Errorf("read sources.json: %w", err)
		}
	} else {
		raw = defaultJSON
		// 写一份默认到磁盘, 方便用户编辑
		_ = os.WriteFile(paths.SourceFile, defaultJSON, 0o644)
	}
	var s Sources
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parse sources.json: %w", err)
	}
	// 兼容: 把单 URL 升级到 URLs 数组
	upgrade := func(es []VersionEntry) []VersionEntry {
		for i := range es {
			if len(es[i].URLs) == 0 && es[i].URL != "" {
				es[i].URLs = []string{es[i].URL}
			}
		}
		return es
	}
	s.Nginx = upgrade(s.Nginx)
	s.Php = upgrade(s.Php)
	s.Mysql = upgrade(s.Mysql)
	s.Postgresql = upgrade(s.Postgresql)
	s.Redis = upgrade(s.Redis)

	// 合并用户自定义版本 (config/custom_sources.json), 追加到内置版本之后,
	// 并标记 Custom=true 让前端能区分.
	if custom, err := LoadCustom(); err == nil && custom != nil {
		s.Nginx = appendCustom(s.Nginx, custom.Nginx)
		s.Php = appendCustom(s.Php, custom.Php)
		s.Mysql = appendCustom(s.Mysql, custom.Mysql)
		s.Postgresql = appendCustom(s.Postgresql, custom.Postgresql)
		s.Redis = appendCustom(s.Redis, custom.Redis)
	}
	return &s, nil
}

func appendCustom(base, custom []VersionEntry) []VersionEntry {
	for _, e := range custom {
		e.Custom = true
		base = append(base, e)
	}
	return base
}

func (s *Sources) Find(kind, version string) *VersionEntry {
	var list []VersionEntry
	switch kind {
	case "nginx":
		list = s.Nginx
	case "php":
		list = s.Php
	case "mysql":
		list = s.Mysql
	case "postgresql":
		list = s.Postgresql
	case "redis":
		list = s.Redis
	}
	for i := range list {
		if list[i].Version == version {
			return &list[i]
		}
	}
	return nil
}
