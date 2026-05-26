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
	"github.com/lly0010/winphp2025/internal/proc"
)

const NginxServiceName = "WinPHPNginx"

type Nginx struct{}

func (Nginx) Name() string { return "nginx" }

func (Nginx) ExePath() string  { return filepath.Join(paths.NginxDir, "nginx.exe") }
func (Nginx) ConfDir() string  { return filepath.Join(paths.NginxDir, "conf") }
func (Nginx) VhostDir() string { return filepath.Join(paths.NginxDir, "conf", "vhosts") }

func (n Nginx) Version() string {
	if _, err := os.Stat(n.ExePath()); err != nil {
		return ""
	}
	out, err := runHidden(n.ExePath(), 3*time.Second, "-v")
	if err != nil {
		return ""
	}
	// nginx writes to stderr; runHidden returns combined
	const prefix = "nginx/"
	if i := strings.Index(out, prefix); i >= 0 {
		s := out[i+len(prefix):]
		end := strings.IndexAny(s, " \r\n\t")
		if end > 0 {
			return s[:end]
		}
		return strings.TrimSpace(s)
	}
	return ""
}

func (n Nginx) Status() Status {
	binOk := false
	if _, err := os.Stat(n.ExePath()); err == nil {
		binOk = true
	}
	svc := ""
	svcInstalled := false
	if ServiceExists(NginxServiceName) {
		svcInstalled = true
		svc = ServiceStatusStr(NginxServiceName)
	}
	running := false
	if svc == "Running" {
		running = true
	}
	if !running && binOk {
		running = proc.HasProcessByPathPrefix("nginx", paths.NginxDir)
	}
	if !running && binOk {
		// 端口兜底
		running = proc.PortListening(80)
	}
	return Status{
		Running: running, Port: 80, Version: n.Version(),
		ServiceInstalled: svcInstalled, ServiceStatus: svc, BinInstalled: binOk,
	}
}

func (n Nginx) Start() error {
	exe := n.ExePath()
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("Nginx 未安装")
	}
	if n.Status().Running {
		return fmt.Errorf("Nginx 已在运行")
	}
	// 配置语法测试
	if out, err := runHidden(exe, 5*time.Second, "-t", "-p", paths.NginxDir); err != nil {
		return fmt.Errorf("nginx -t 失败: %v\n%s", err, out)
	}

	if ServiceExists(NginxServiceName) {
		if err := StartService(NginxServiceName); err != nil {
			return fmt.Errorf("service start: %w", err)
		}
	} else {
		cmd := exec.Command(exe, "-p", paths.NginxDir)
		cmd.Dir = paths.NginxDir
		hideWindow(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		_ = cmd.Process.Release()
	}
	time.Sleep(700 * time.Millisecond)
	logger.Info("Nginx 启动")
	return nil
}

func (n Nginx) Stop() error {
	if ServiceExists(NginxServiceName) {
		_ = StopService(NginxServiceName)
	}
	exe := n.ExePath()
	if _, err := os.Stat(exe); err == nil && proc.HasProcessByPathPrefix("nginx", paths.NginxDir) {
		_, _ = runHidden(exe, 5*time.Second, "-s", "stop", "-p", paths.NginxDir)
	}
	time.Sleep(400 * time.Millisecond)
	// 强杀残留
	killByPathPrefix("nginx", paths.NginxDir)
	logger.Info("Nginx 已停止")
	return nil
}

func (n Nginx) Restart() error {
	_ = n.Stop()
	time.Sleep(300 * time.Millisecond)
	return n.Start()
}

func (n Nginx) Reload() error {
	exe := n.ExePath()
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("Nginx 未安装")
	}
	if out, err := runHidden(exe, 5*time.Second, "-t", "-p", paths.NginxDir); err != nil {
		return fmt.Errorf("nginx -t: %v\n%s", err, out)
	}
	if !proc.HasProcessByPathPrefix("nginx", paths.NginxDir) {
		return fmt.Errorf("Nginx 未运行")
	}
	_, _ = runHidden(exe, 5*time.Second, "-s", "reload", "-p", paths.NginxDir)
	logger.Info("Nginx 已 reload")
	return nil
}

// InitConfig 安装/更新 nginx 后写默认配置
func (Nginx) InitConfig() error {
	if _, err := os.Stat(paths.NginxDir); err != nil {
		return err
	}
	confDir := filepath.Join(paths.NginxDir, "conf")
	if err := os.MkdirAll(filepath.Join(confDir, "vhosts"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(paths.NginxDir, "logs"), 0o755); err != nil {
		return err
	}
	tpl, _ := readTemplate("nginx.conf", defaultNginxConf)
	conf := strings.ReplaceAll(tpl, "##WWW_ROOT##", filepath.ToSlash(paths.WwwDir))
	if err := os.WriteFile(filepath.Join(confDir, "nginx.conf"), []byte(conf), 0o644); err != nil {
		return err
	}
	// 默认首页
	defDir := filepath.Join(paths.WwwDir, "default")
	_ = os.MkdirAll(defDir, 0o755)
	idx := filepath.Join(defDir, "index.php")
	if _, err := os.Stat(idx); err != nil {
		_ = os.WriteFile(idx, []byte("<?php\necho '<h1>WinPHP - It works!</h1>';\necho '<p>PHP ' . phpversion() . '</p>';\necho '<a href=\"phpinfo.php\">phpinfo</a>';\n"), 0o644)
	}
	info := filepath.Join(defDir, "phpinfo.php")
	if _, err := os.Stat(info); err != nil {
		_ = os.WriteFile(info, []byte("<?php phpinfo();\n"), 0o644)
	}
	logger.Info("Nginx 配置初始化完成")
	return nil
}

const defaultNginxConf = `worker_processes auto;
events { worker_connections 1024; }
http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout 65;
    server_tokens off;
    client_max_body_size 64m;
    gzip on;
    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"';
    access_log logs/access.log main;
    error_log  logs/error.log;
    server {
        listen 80 default_server;
        server_name localhost;
        root "##WWW_ROOT##/default";
        index index.html index.htm index.php;
        location / { try_files $uri $uri/ /index.php?$query_string; }
        location ~ \.php$ {
            fastcgi_pass 127.0.0.1:9000;
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
            include fastcgi_params;
        }
        location ~ /\.ht { deny all; }
    }
    include vhosts/*.conf;
}
`
