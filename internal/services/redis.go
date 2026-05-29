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
	"github.com/lly0010/winphp2025/internal/portcheck"
	"github.com/lly0010/winphp2025/internal/proc"
	"github.com/lly0010/winphp2025/internal/state"
)

const RedisServiceName = "WinPHPRedis"

type Redis struct{}

func (Redis) Name() string         { return "redis" }
func (Redis) ExePath() string      { return filepath.Join(paths.RedisDir, "redis-server.exe") }
func (Redis) CliPath() string      { return filepath.Join(paths.RedisDir, "redis-cli.exe") }
func (Redis) ConfPath() string     { return filepath.Join(paths.RedisDir, "redis.windows.conf") }

func (r Redis) Version() string {
	if _, err := os.Stat(r.ExePath()); err != nil {
		return ""
	}
	out, err := runHidden(r.ExePath(), 3*time.Second, "--version")
	if err == nil {
		// "Redis server v=5.0.14.1 sha=..." 抓 v= 后的版本号
		if i := strings.Index(out, "v="); i >= 0 {
			s := out[i+2:]
			end := strings.IndexAny(s, " \r\n\t")
			if end > 0 {
				return s[:end]
			}
		}
	}
	// redis-server.exe 跑不起来 (常见: 缺 VC++ Redist) 或输出异常, 回退 state 里的版本号
	if st := state.Load(); st.RedisVersion != "" {
		return st.RedisVersion
	}
	return ""
}

func (r Redis) Status() Status {
	binOk := false
	if _, err := os.Stat(r.ExePath()); err == nil {
		binOk = true
	}
	svcInstalled := ServiceExists(RedisServiceName)
	svc := ""
	if svcInstalled {
		svc = ServiceStatusStr(RedisServiceName)
	}
	running := svc == "Running"
	if !running && binOk {
		running = proc.HasProcessByPathPrefix("redis-server", paths.RedisDir)
	}
	if !running && binOk {
		running = proc.PortListening(6379)
	}
	return Status{
		Running: running, Port: 6379, Version: r.Version(),
		ServiceInstalled: svcInstalled, ServiceStatus: svc, BinInstalled: binOk,
	}
}

func (r Redis) Start() error {
	if _, err := os.Stat(r.ExePath()); err != nil {
		return fmt.Errorf("Redis 未安装")
	}
	if r.Status().Running {
		return fmt.Errorf("Redis 已运行")
	}
	// 自我修复 redis.windows.conf
	if _, err := os.Stat(r.ConfPath()); err != nil {
		logger.Warn("redis.windows.conf 不存在, 自动重新生成")
		if e := (Redis{}).InitConfig(); e != nil {
			return fmt.Errorf("Redis 配置不存在, 自动生成失败: %v", e)
		}
	}
	if proc.PortListening(6379) {
		return fmt.Errorf("端口 6379 已被占用. %s", portcheck.Diagnose(6379).Diagnosis)
	}
	if ServiceExists(RedisServiceName) {
		return StartService(RedisServiceName)
	}
	conf := r.ConfPath()
	args := []string{}
	if _, err := os.Stat(conf); err == nil {
		args = []string{conf}
	}
	cmd := exec.Command(r.ExePath(), args...)
	cmd.Dir = paths.RedisDir
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	time.Sleep(800 * time.Millisecond)
	logger.Info("Redis 启动")
	return nil
}

func (r Redis) Stop() error {
	if ServiceExists(RedisServiceName) {
		_ = StopService(RedisServiceName)
	}
	// 优雅关停: redis-cli shutdown (有密码就带 -a)
	if _, err := os.Stat(r.CliPath()); err == nil {
		args := []string{"-h", "127.0.0.1", "-p", "6379"}
		if pwd := r.Password(); pwd != "" {
			args = append(args, "-a", pwd)
		}
		args = append(args, "shutdown", "nosave")
		_, _ = runHidden(r.CliPath(), 5*time.Second, args...)
	}
	time.Sleep(500 * time.Millisecond)
	killByPathPrefix("redis-server", paths.RedisDir)
	logger.Info("Redis 已停止")
	return nil
}

// Password 从 redis.windows.conf 里抓当前 requirepass. 空串=无密码.
func (r Redis) Password() string {
	b, err := os.ReadFile(r.ConfPath())
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "#") {
			continue
		}
		if strings.HasPrefix(l, "requirepass") {
			rest := strings.TrimSpace(strings.TrimPrefix(l, "requirepass"))
			rest = strings.Trim(rest, `"' `)
			return rest
		}
	}
	return ""
}

// SetPassword 改 Redis 访问密码.
// 空串 = 移除密码 (清掉 conf 里的 requirepass 行).
// 写完 conf 后, 如果 Redis 正在运行还会通过 CONFIG SET 立即生效, 不需要重启.
func (r Redis) SetPassword(newPwd string) error {
	if strings.ContainsAny(newPwd, "\n\r\"") {
		return fmt.Errorf("密码不能包含换行或双引号")
	}
	confPath := r.ConfPath()
	// 自我修复: 配置不存在就先生成
	if _, err := os.Stat(confPath); err != nil {
		if e := (Redis{}).InitConfig(); e != nil {
			return fmt.Errorf("redis 配置不存在, 自动生成失败: %v", e)
		}
	}
	oldPwd := r.Password()
	text, err := readFileAll(confPath)
	if err != nil {
		return err
	}
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, l := range lines {
		body := strings.TrimLeft(strings.TrimSpace(l), "# \t")
		if strings.HasPrefix(body, "requirepass") {
			continue
		}
		kept = append(kept, l)
	}
	if newPwd != "" {
		kept = append(kept, `requirepass "`+newPwd+`"`)
	}
	if err := os.WriteFile(confPath, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		return err
	}
	// 运行中就直接 CONFIG SET 一把, 立即生效
	if proc.HasProcessByPathPrefix("redis-server", paths.RedisDir) || proc.PortListening(6379) {
		if _, err := os.Stat(r.CliPath()); err == nil {
			args := []string{"-h", "127.0.0.1", "-p", "6379"}
			if oldPwd != "" {
				args = append(args, "-a", oldPwd)
			}
			args = append(args, "CONFIG", "SET", "requirepass", newPwd)
			if out, err := runHidden(r.CliPath(), 5*time.Second, args...); err != nil {
				logger.Warn("Redis CONFIG SET requirepass 失败 (重启 Redis 后仍会生效): %v\n%s", err, out)
			}
		}
	}
	if newPwd == "" {
		logger.Info("Redis 密码已清除")
	} else {
		logger.Info("Redis 密码已更新")
	}
	return nil
}

func (r Redis) Restart() error {
	_ = r.Stop()
	time.Sleep(500 * time.Millisecond)
	return r.Start()
}

const defaultRedisConf = `# WinPHP 默认 Redis 配置 (开发模式)
bind 127.0.0.1
port 6379
protected-mode yes
tcp-backlog 511
timeout 0
tcp-keepalive 300
daemonize no
loglevel notice
logfile "logs/redis.log"
databases 16
save 900 1
save 300 10
save 60 10000
dbfilename dump.rdb
dir "./"
maxmemory-policy noeviction
`

func (Redis) InitConfig() error {
	if _, err := os.Stat(paths.RedisDir); err != nil {
		return err
	}
	confPath := filepath.Join(paths.RedisDir, "redis.windows.conf")
	tpl, _ := readTemplate("redis.conf", defaultRedisConf)
	if err := os.WriteFile(confPath, []byte(tpl), 0o644); err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Join(paths.RedisDir, "logs"), 0o755)
	logger.Info("Redis 配置初始化完成")
	return nil
}
