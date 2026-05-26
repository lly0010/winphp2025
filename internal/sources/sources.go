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
	raw        map[string]interface{} `json:"-"`
}

type NssmEntry struct {
	Version  string   `json:"version"`
	URL      string   `json:"url"`
	Mirrors  []string `json:"mirrors,omitempty"`
	ExeInZip string   `json:"exeInZip"`
}

type VersionEntry struct {
	Version   string   `json:"version"`
	URLs      []string `json:"urls"`
	URL       string   `json:"url,omitempty"` // 兼容旧字段
	RootInZip string   `json:"rootInZip"`
	VsTag     string   `json:"vs,omitempty"` // PHP: vs16 / vs17 / vc15
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
	return &s, nil
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
	}
	for i := range list {
		if list[i].Version == version {
			return &list[i]
		}
	}
	return nil
}
