package sites

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lly0010/winphp2025/internal/hosts"
	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/state"
)

const defaultVhost = `server {
    listen       ##PORT## ;
    server_name  ##SERVER_NAME## ;
    root         "##ROOT##" ;
    index        index.html index.htm index.php ;

    access_log   logs/##SITE##.access.log main ;
    error_log    logs/##SITE##.error.log ;

    location / {
        try_files $uri $uri/ /index.php?$query_string ;
    }

    location ~ \.php$ {
        fastcgi_pass   127.0.0.1:9000 ;
        fastcgi_index  index.php ;
        fastcgi_param  SCRIPT_FILENAME  $document_root$fastcgi_script_name ;
        include        fastcgi_params ;
    }

    location ~ /\.(ht|git|svn) { deny all; }
}
`

type AddSiteInput struct {
	Name       string `json:"name"`
	ServerName string `json:"serverName"`
	Root       string `json:"root"`
	Port       int    `json:"port"`
	Template   string `json:"template"` // "php" / "laravel" / "wordpress" / "static"
	AddHosts   bool   `json:"addHosts"`
}

func Add(in AddSiteInput) error {
	if in.Name == "" {
		return fmt.Errorf("站点名不能为空")
	}
	if in.ServerName == "" {
		return fmt.Errorf("域名不能为空")
	}
	if in.Port <= 0 {
		in.Port = 80
	}
	if in.Root == "" {
		in.Root = filepath.Join(paths.WwwDir, in.Name)
	}
	if err := os.MkdirAll(in.Root, 0o755); err != nil {
		return err
	}

	// 模板处理
	vhostRoot := in.Root
	switch in.Template {
	case "laravel":
		vhostRoot = filepath.Join(in.Root, "public")
		_ = os.MkdirAll(vhostRoot, 0o755)
		idx := filepath.Join(vhostRoot, "index.php")
		if _, err := os.Stat(idx); err != nil {
			_ = os.WriteFile(idx, []byte("<?php\n// Laravel public 占位. 请用 composer create-project laravel/laravel 覆盖上层目录.\necho 'Laravel public placeholder';\n"), 0o644)
		}
	case "wordpress":
		readme := filepath.Join(in.Root, "README.txt")
		if _, err := os.Stat(readme); err != nil {
			_ = os.WriteFile(readme, []byte("请将 WordPress (https://cn.wordpress.org/latest-zh_CN.zip) 解压到本目录, 然后访问站点完成安装.\n"), 0o644)
		}
	case "static":
		idx := filepath.Join(in.Root, "index.html")
		if _, err := os.Stat(idx); err != nil {
			html := fmt.Sprintf("<!DOCTYPE html><html><head><meta charset='UTF-8'><title>%s</title></head><body><h1>%s</h1><p>纯静态站点 (WinPHP)</p></body></html>", in.ServerName, in.ServerName)
			_ = os.WriteFile(idx, []byte(html), 0o644)
		}
	default: // php
		idx := filepath.Join(in.Root, "index.php")
		if _, err := os.Stat(idx); err != nil {
			php := fmt.Sprintf("<?php\necho '<h1>%s</h1>';\necho '<p>PHP ' . phpversion() . ' - WinPHP</p>';\necho '<p>Document Root: ' . $_SERVER['DOCUMENT_ROOT'] . '</p>';\n", in.ServerName)
			_ = os.WriteFile(idx, []byte(php), 0o644)
		}
	}

	// 生成 vhost
	vhostDir := filepath.Join(paths.NginxDir, "conf", "vhosts")
	if err := os.MkdirAll(vhostDir, 0o755); err != nil {
		return err
	}
	vhost := strings.NewReplacer(
		"##SITE##", in.Name,
		"##SERVER_NAME##", in.ServerName,
		"##ROOT##", filepath.ToSlash(vhostRoot),
		"##PORT##", fmt.Sprintf("%d", in.Port),
	).Replace(defaultVhost)
	if err := os.WriteFile(filepath.Join(vhostDir, in.Name+".conf"), []byte(vhost), 0o644); err != nil {
		return err
	}

	// 写 sites.json
	ss := state.Sites()
	// 去重
	out := ss[:0]
	for _, s := range ss {
		if s.Name != in.Name {
			out = append(out, s)
		}
	}
	out = append(out, state.Site{
		Name:       in.Name,
		ServerName: in.ServerName,
		Root:       vhostRoot,
		Port:       in.Port,
		Template:   in.Template,
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	})
	if err := state.SaveSites(out); err != nil {
		return err
	}

	if in.AddHosts && in.ServerName != "localhost" {
		if err := hosts.Add(in.ServerName); err != nil {
			logger.Warn("写入 hosts 失败: %v", err)
		}
	}
	logger.Info("已添加站点 %s (%s -> %s)", in.Name, in.ServerName, vhostRoot)
	return nil
}

func Remove(name string, removeHosts bool) error {
	ss := state.Sites()
	var target *state.Site
	out := ss[:0]
	for i := range ss {
		if ss[i].Name == name {
			target = &ss[i]
			continue
		}
		out = append(out, ss[i])
	}
	if target == nil {
		return fmt.Errorf("站点 %s 不存在", name)
	}
	vf := filepath.Join(paths.NginxDir, "conf", "vhosts", name+".conf")
	_ = os.Remove(vf)
	if err := state.SaveSites(out); err != nil {
		return err
	}
	if removeHosts && target.ServerName != "localhost" {
		_ = hosts.Remove(target.ServerName)
	}
	logger.Info("已删除站点 %s", name)
	return nil
}

func List() []state.Site {
	return state.Sites()
}
