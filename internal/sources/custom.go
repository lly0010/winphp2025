package sources

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/lly0010/winphp2025/internal/paths"
)

const customFileName = "custom_sources.json"

// Custom 用户自定义的版本源 (不污染内置 sources_default.json).
type Custom struct {
	Nginx      []VersionEntry `json:"nginx"`
	Php        []VersionEntry `json:"php"`
	Mysql      []VersionEntry `json:"mysql"`
	Postgresql []VersionEntry `json:"postgresql"`
	Redis      []VersionEntry `json:"redis"`
}

func customPath() string {
	return filepath.Join(paths.ConfigDir, customFileName)
}

func LoadCustom() (*Custom, error) {
	c := &Custom{}
	p := customPath()
	if _, err := os.Stat(p); err != nil {
		return c, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal(b, c)
	return c, nil
}

func SaveCustom(c *Custom) error {
	p := customPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func (c *Custom) Get(kind string) []VersionEntry {
	switch kind {
	case "nginx":
		return c.Nginx
	case "php":
		return c.Php
	case "mysql":
		return c.Mysql
	case "postgresql", "postgres":
		return c.Postgresql
	case "redis":
		return c.Redis
	}
	return nil
}

func (c *Custom) Set(kind string, v []VersionEntry) {
	switch kind {
	case "nginx":
		c.Nginx = v
	case "php":
		c.Php = v
	case "mysql":
		c.Mysql = v
	case "postgresql", "postgres":
		c.Postgresql = v
	case "redis":
		c.Redis = v
	}
}

// Upsert 添加或覆盖同 version 的条目.
func (c *Custom) Upsert(kind string, e VersionEntry) {
	e.Custom = true
	list := c.Get(kind)
	for i := range list {
		if list[i].Version == e.Version {
			list[i] = e
			c.Set(kind, list)
			return
		}
	}
	c.Set(kind, append(list, e))
}

// Remove 按 version 删除一条.
func (c *Custom) Remove(kind, version string) bool {
	list := c.Get(kind)
	out := list[:0]
	removed := false
	for _, e := range list {
		if e.Version == version {
			removed = true
			continue
		}
		out = append(out, e)
	}
	c.Set(kind, out)
	return removed
}
