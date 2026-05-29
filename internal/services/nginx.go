package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/portcheck"
	"github.com/lly0010/winphp2025/internal/proc"
	"github.com/lly0010/winphp2025/internal/state"
	"github.com/lly0010/winphp2025/internal/winshort"
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
	if err == nil {
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
	}
	// nginx.exe 跑不起来 (常见: 缺 VC++ Redist) 或输出异常, 回退 state 里的版本号
	if st := state.Load(); st.NginxVersion != "" {
		return st.NginxVersion
	}
	return ""
}

// CurrentPort 读当前 nginx.conf 里 default_server 的 listen 端口. 默认 80.
func (n Nginx) CurrentPort() int { return detectNginxPort() }

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
	port := detectNginxPort()
	running := false
	if svc == "Running" {
		running = true
	}
	if !running && binOk {
		running = proc.HasProcessByPathPrefix("nginx", paths.NginxDir)
	}
	if !running && binOk {
		// 端口兜底 (用 conf 里实际的端口)
		running = proc.PortListening(port)
	}
	return Status{
		Running: running, Port: port, Version: n.Version(),
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
	// 自我修复: 启动前检查 conf/nginx.conf 是否存在, 不存在就重新生成.
	// 防止 zip 解压异常 / 用户手动删配置 / 升级时遗留不完整状态.
	confFile := filepath.Join(paths.NginxDir, "conf", "nginx.conf")
	if _, err := os.Stat(confFile); err != nil {
		logger.Warn("nginx.conf 不存在 (%s), 自动重新生成默认配置", confFile)
		if e := n.InitConfig(); e != nil {
			return fmt.Errorf("nginx.conf 不存在, 自动生成也失败: %v\n请重新点 '安装/切换版本' 重装", e)
		}
		if _, err := os.Stat(confFile); err != nil {
			return fmt.Errorf("nginx.conf 仍不存在 (%s), 请检查 bin/nginx/conf 目录权限或重装", confFile)
		}
	}
	// 配置语法测试.
	// 注意: 不传 -p 参数, 改用工作目录 (cmd.Dir = winshort), 让 nginx 自己
	// 用 cwd 作 prefix. nginx 对 -p 后路径的混合斜杠拼接有 bug
	// ("bin\\nginx" + "conf/nginx.conf" => "bin\\nginx\\conf/nginx.conf"),
	// 走 cwd 完全规避.
	nginxShort := winshort.ShortIfNeeded(paths.NginxDir)
	if out, err := runHiddenIn(nginxShort, exe, 5*time.Second, "-t"); err != nil {
		return fmt.Errorf("nginx -t 失败: %v\n%s", err, out)
	}

	if ServiceExists(NginxServiceName) {
		if err := StartService(NginxServiceName); err != nil {
			return fmt.Errorf("service start: %w", err)
		}
	} else {
		// 用工作目录代替 -p 解决中文目录启动失败 + 路径斜杠混合 bug
		cmd := exec.Command(exe)
		cmd.Dir = nginxShort
		hideWindow(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		_ = cmd.Process.Release()
	}
	time.Sleep(700 * time.Millisecond)
	// 校验是否真的起来了; 没起来调端口诊断给友好提示
	if !n.Status().Running {
		// 探测当前 listen 端口 (默认 80, 也可能是 vhost 改成的其他端口)
		port := detectNginxPort()
		diag := portcheck.Diagnose(port)
		return fmt.Errorf(
			"Nginx 启动后立即退出 (常见: 端口绑定失败). %s\n\n"+
				"建议操作:\n"+
				"  • 改 nginx.conf 把 'listen %d' 换成空闲端口 (如 8080), 浏览器用 http://localhost:8080\n"+
				"  • 或在管理员 CMD 跑 'net stop winnat' 释放 Windows 预留端口, 再 'net start winnat'\n"+
				"  • 查看 bin/nginx/logs/error.log 获取详细错误",
			diag.Diagnosis, port,
		)
	}
	logger.Info("Nginx 启动")
	return nil
}

// SetDefaultPort 改写 nginx.conf 里 default_server 那行的监听端口.
// 只动 default_server 那行 (主默认站), 各 vhost 的 listen 端口不变.
// 写完不会 reload, 由调用方决定.
func (n Nginx) SetDefaultPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("端口范围 1-65535")
	}
	confFile := filepath.Join(paths.NginxDir, "conf", "nginx.conf")
	b, err := os.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("读 nginx.conf 失败: %w", err)
	}
	// 支持两种形式:
	//   listen 80 default_server;
	//   listen 127.0.0.1:80 default_server;
	// 不动各 vhost 的非 default_server listen 行.
	re := regexp.MustCompile(`(?m)^(\s*listen\s+(?:[^\s;:]+:)?)\d+(\s+default_server.*)$`)
	replaced := re.ReplaceAllString(string(b), fmt.Sprintf("${1}%d${2}", port))
	if replaced == string(b) {
		return fmt.Errorf("nginx.conf 里没找到 'listen <port> default_server' 行 (可能被手动改过), 请直接编辑 nginx.conf 改 listen")
	}
	if err := os.WriteFile(confFile, []byte(replaced), 0o644); err != nil {
		return fmt.Errorf("写 nginx.conf 失败: %w", err)
	}
	return nil
}

// detectNginxPort 从 nginx.conf 抓 'listen NNN' 第一个数字, 找不到就当 80.
func detectNginxPort() int {
	b, err := os.ReadFile(filepath.Join(paths.NginxDir, "conf", "nginx.conf"))
	if err != nil {
		return 80
	}
	for _, line := range strings.Split(string(b), "\n") {
		l := strings.TrimSpace(line)
		if !strings.HasPrefix(l, "listen") {
			continue
		}
		// listen 80; / listen 80 default_server; / listen 127.0.0.1:80;
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		v := strings.TrimSuffix(fields[1], ";")
		if i := strings.LastIndex(v, ":"); i >= 0 {
			v = v[i+1:]
		}
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		if err == nil && n > 0 {
			return n
		}
	}
	return 80
}

func (n Nginx) Stop() error {
	if ServiceExists(NginxServiceName) {
		_ = StopService(NginxServiceName)
	}
	exe := n.ExePath()
	if _, err := os.Stat(exe); err == nil && proc.HasProcessByPathPrefix("nginx", paths.NginxDir) {
		_, _ = runHiddenIn(winshort.ShortIfNeeded(paths.NginxDir), exe, 5*time.Second, "-s", "stop")
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
	wd := winshort.ShortIfNeeded(paths.NginxDir)
	if out, err := runHiddenIn(wd, exe, 5*time.Second, "-t"); err != nil {
		return fmt.Errorf("nginx -t: %v\n%s", err, out)
	}
	if !proc.HasProcessByPathPrefix("nginx", paths.NginxDir) {
		return fmt.Errorf("Nginx 未运行")
	}
	_, _ = runHiddenIn(wd, exe, 5*time.Second, "-s", "reload")
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
	// 用短路径避免中文 root 启动失败
	wwwShort := filepath.ToSlash(winshort.ShortIfNeeded(paths.WwwDir))
	conf := strings.ReplaceAll(tpl, "##WWW_ROOT##", wwwShort)
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
