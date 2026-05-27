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
	"github.com/lly0010/winphp2025/internal/winshort"
)

// vhost 模板. 占位符:
//   ##SITE## / ##SERVER_NAME## / ##ROOT## / ##PORT##
//   ##REWRITE_BLOCK## - 伪静态规则 (location / { ... }), 不同框架不同
const defaultVhost = `server {
    listen       ##PORT## ;
    server_name  ##SERVER_NAME## ;
    root         "##ROOT##" ;
    index        index.html index.htm index.php ;

    access_log   logs/##SITE##.access.log main ;
    error_log    logs/##SITE##.error.log ;

##REWRITE_BLOCK##

    location ~ \.php$ {
        fastcgi_pass   127.0.0.1:9000 ;
        fastcgi_index  index.php ;
        fastcgi_param  SCRIPT_FILENAME  $document_root$fastcgi_script_name ;
        include        fastcgi_params ;
    }

    location ~ /\.(ht|git|svn) { deny all; }
}
`

// rewriteBlock 按伪静态类型生成 nginx location 块.
func rewriteBlock(kind string) string {
	switch kind {
	case "none":
		return `    location / {
        try_files $uri $uri/ =404 ;
    }`
	case "thinkphp":
		return `    # 伪静态: ThinkPHP
    location / {
        if (!-e $request_filename) {
            rewrite ^(.*)$ /index.php?s=$1 last;
            break;
        }
        try_files $uri $uri/ /index.php?$query_string ;
    }`
	case "discuz":
		return `    # 伪静态: Discuz!
    location / {
        rewrite ^([^\.]*)/topic-(.+)\.html$ $1/portal.php?mod=topic&topic=$2 last;
        rewrite ^([^\.]*)/article-([0-9]+)-([0-9]+)\.html$ $1/portal.php?mod=view&aid=$2&page=$3 last;
        rewrite ^([^\.]*)/forum-(\w+)-([0-9]+)\.html$ $1/forumdisplay.php?fid=$2&page=$3 last;
        rewrite ^([^\.]*)/thread-([0-9]+)-([0-9]+)-([0-9]+)\.html$ $1/viewthread.php?tid=$2&extra=page%3D$4&page=$3 last;
        rewrite ^([^\.]*)/group-([0-9]+)-([0-9]+)\.html$ $1/forumdisplay.php?fid=$2&page=$3 last;
        rewrite ^([^\.]*)/space-(username|uid)-(.+)\.html$ $1/space.php?$2=$3 last;
        rewrite ^([^\.]*)/blog-([0-9]+)-([0-9]+)\.html$ $1/space.php?uid=$2&do=blog&id=$3 last;
        rewrite ^([^\.]*)/(fid|tid)-([0-9]+)\.html$ $1/index.php?$2=$3 last;
        try_files $uri $uri/ /index.php?$query_string ;
    }`
	case "ecshop":
		return `    # 伪静态: ECShop
    location / {
        rewrite "^/index.html" /index.php last;
        rewrite "^/category$" /index.php last;
        rewrite "^/feed-c([0-9]+).xml$" /feed.php?cat=$1 last;
        rewrite "^/feed-b([0-9]+).xml$" /feed.php?brand=$1 last;
        rewrite "^/feed.xml$" /feed.php last;
        rewrite "^/category-([0-9]+)(.*)$" /category.php?id=$1$2 last;
        rewrite "^/goods-([0-9]+)(.*)$" /goods.php?id=$1 last;
        rewrite "^/article_cat-([0-9]+)(.*)$" /article_cat.php?id=$1$2 last;
        rewrite "^/article-([0-9]+)(.*)$" /article.php?id=$1 last;
        rewrite "^/brand-([0-9]+)(.*)$" /brand.php?id=$1$2 last;
        rewrite "^/tag-(.*)$" /search.php?keywords=$1 last;
        rewrite "^/snatch-([0-9]+)\.html$" /snatch.php?id=$1 last;
        rewrite "^/group_buy-([0-9]+)\.html$" /group_buy.php?act=view&id=$1 last;
        rewrite "^/auction-([0-9]+)\.html$" /auction.php?act=view&id=$1 last;
        rewrite "^/exchange-id([0-9]+)(.*)$" /exchange.php?id=$1$2 last;
        rewrite "^/exchange-([0-9]+)(.*)$" /exchange.php?cat_id=$1$2 last;
        try_files $uri $uri/ /index.php?$query_string ;
    }`
	}
	// "default" / 默认 PHP 框架 (Laravel / WordPress / Yii ...)
	return `    location / {
        try_files $uri $uri/ /index.php?$query_string ;
    }`
}

type AddSiteInput struct {
	Name       string `json:"name"`
	ServerName string `json:"serverName"`
	Root       string `json:"root"`
	Port       int    `json:"port"`
	Template   string `json:"template"` // "php" / "laravel" / "wordpress" / "static"
	Rewrite    string `json:"rewrite"`  // "default" / "thinkphp" / "discuz" / "ecshop" / "none"
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
	// 静态站点强制 none 伪静态
	rewriteKind := in.Rewrite
	if rewriteKind == "" {
		rewriteKind = "default"
	}
	if in.Template == "static" {
		rewriteKind = "none"
	}

	// 用 Windows 短路径避免中文目录导致 nginx 启动失败
	vhost := strings.NewReplacer(
		"##SITE##", in.Name,
		"##SERVER_NAME##", in.ServerName,
		"##ROOT##", filepath.ToSlash(winshort.Short(vhostRoot)),
		"##PORT##", fmt.Sprintf("%d", in.Port),
		"##REWRITE_BLOCK##", rewriteBlock(rewriteKind),
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
		Rewrite:    rewriteKind,
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
