package services

import (
	"context"
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
	"github.com/lly0010/winphp2025/internal/textenc"
)

const MysqlServiceName = "WinPHPMySQL"

type MySQL struct{}

func (MySQL) Name() string         { return "mysql" }
func (MySQL) ExePath() string      { return filepath.Join(paths.MysqlDir, "bin", "mysqld.exe") }
func (MySQL) ClientPath() string   { return filepath.Join(paths.MysqlDir, "bin", "mysql.exe") }
func (MySQL) AdminPath() string    { return filepath.Join(paths.MysqlDir, "bin", "mysqladmin.exe") }
func (MySQL) IniPath() string      { return filepath.Join(paths.MysqlDir, "my.ini") }
func (MySQL) DataPath() string     { return filepath.Join(paths.MysqlDir, "data") }

func (m MySQL) Version() string {
	if _, err := os.Stat(m.ExePath()); err != nil {
		return ""
	}
	out, err := runHidden(m.ExePath(), 3*time.Second, "--version")
	if err == nil {
		if i := strings.Index(out, "Ver "); i >= 0 {
			s := out[i+4:]
			end := strings.IndexAny(s, " \r\n\t-")
			if end > 0 {
				return s[:end]
			}
		}
	}
	// mysqld.exe 跑不起来 (常见: 缺 VC++ Redist) 或输出异常, 回退 state 里的版本号
	if st := state.Load(); st.MysqlVersion != "" {
		return st.MysqlVersion
	}
	return ""
}

func (m MySQL) Status() Status {
	binOk := false
	if _, err := os.Stat(m.ExePath()); err == nil {
		binOk = true
	}
	svcInstalled := ServiceExists(MysqlServiceName)
	svc := ""
	if svcInstalled {
		svc = ServiceStatusStr(MysqlServiceName)
	}
	running := svc == "Running"
	if !running && binOk {
		running = proc.HasProcessByPathPrefix("mysqld", paths.MysqlDir)
	}
	if !running && binOk {
		running = proc.PortListening(3306)
	}
	return Status{
		Running: running, Port: 3306, Version: m.Version(),
		ServiceInstalled: svcInstalled, ServiceStatus: svc, BinInstalled: binOk,
	}
}

func (m MySQL) initData() error {
	dataDir := m.DataPath()
	if _, err := os.Stat(filepath.Join(dataDir, "mysql")); err == nil {
		return nil
	}
	_ = os.RemoveAll(dataDir)
	logger.Info("MySQL 正在初始化 data 目录 (1-2 分钟)...")
	out, err := runHidden(m.ExePath(), 3*time.Minute,
		"--defaults-file="+m.IniPath(), "--initialize-insecure", "--console")
	if err != nil {
		return fmt.Errorf("mysql initialize: %v\n%s", err, out)
	}
	st := state.Load()
	st.MysqlInited = true
	_ = state.Save(st)
	logger.Info("MySQL 初始化完成, root 密码为空")
	return nil
}

func (m MySQL) Start() error {
	exe := m.ExePath()
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("MySQL 未安装")
	}
	if m.Status().Running {
		return fmt.Errorf("MySQL 已运行")
	}
	// 自我修复 my.ini
	if _, err := os.Stat(m.IniPath()); err != nil {
		logger.Warn("my.ini 不存在, 自动重新生成")
		if e := m.InitConfig(); e != nil {
			return fmt.Errorf("my.ini 不存在, 自动生成失败: %v", e)
		}
	}
	st := state.Load()
	if !st.MysqlInited {
		if err := m.initData(); err != nil {
			return err
		}
	}
	if proc.PortListening(3306) {
		return fmt.Errorf("端口 3306 已被占用. %s", portcheck.Diagnose(3306).Diagnosis)
	}
	if ServiceExists(MysqlServiceName) {
		return StartService(MysqlServiceName)
	}
	cmd := exec.Command(exe, "--defaults-file="+m.IniPath())
	cmd.Dir = paths.MysqlDir
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	time.Sleep(2 * time.Second)
	logger.Info("MySQL 启动")
	return nil
}

func (m MySQL) Stop() error {
	if ServiceExists(MysqlServiceName) {
		_ = StopService(MysqlServiceName)
	}
	if _, err := os.Stat(m.AdminPath()); err == nil {
		// 优雅关停
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, m.AdminPath(), "-u", "root", "--protocol=tcp", "-h", "127.0.0.1", "shutdown")
		hideWindow(cmd)
		_ = cmd.Run()
	}
	time.Sleep(800 * time.Millisecond)
	killByPathPrefix("mysqld", paths.MysqlDir)
	logger.Info("MySQL 已停止")
	return nil
}

func (m MySQL) Restart() error {
	_ = m.Stop()
	time.Sleep(500 * time.Millisecond)
	return m.Start()
}

const defaultMyIni = `[mysqld]
basedir = "##MYSQL_DIR##"
datadir = "##MYSQL_DIR##/data"
tmpdir  = "##MYSQL_DIR##/tmp"
port    = 3306
bind-address = 127.0.0.1
max_connections = 200
max_allowed_packet = 64M
character-set-server = utf8mb4
collation-server     = utf8mb4_unicode_ci
default-authentication-plugin = mysql_native_password
default-storage-engine = InnoDB
innodb_buffer_pool_size = 256M
innodb_log_file_size = 64M
innodb_flush_log_at_trx_commit = 2
innodb_file_per_table = 1
log-error = "##MYSQL_DIR##/logs/error.log"
sql_mode = "STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION"

[client]
port = 3306
default-character-set = utf8mb4

[mysql]
default-character-set = utf8mb4
`

func (m MySQL) InitConfig() error {
	if _, err := os.Stat(paths.MysqlDir); err != nil {
		return err
	}
	tpl, _ := readTemplate("my.ini", defaultMyIni)
	conf := strings.ReplaceAll(tpl, "##MYSQL_DIR##", filepath.ToSlash(paths.MysqlDir))
	if err := os.WriteFile(m.IniPath(), []byte(conf), 0o644); err != nil {
		return err
	}
	for _, d := range []string{"logs", "tmp"} {
		_ = os.MkdirAll(filepath.Join(paths.MysqlDir, d), 0o755)
	}
	logger.Info("MySQL 配置初始化完成")
	return nil
}

func (m MySQL) SetRootPassword(newPwd string) error {
	if !m.Status().Running {
		return fmt.Errorf("MySQL 未运行")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, m.AdminPath(), "-u", "root", "--protocol=tcp", "-h", "127.0.0.1", "password", newPwd)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("修改失败: %v\n%s", err, textenc.ToUTF8(out))
	}
	return nil
}

// CreateDatabase via mysql client
func (m MySQL) CreateDatabase(name, rootPwd string) error {
	if !m.Status().Running {
		return fmt.Errorf("MySQL 未运行")
	}
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, name)
	stmt := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", sanitized)
	args := []string{"-u", "root", "--protocol=tcp", "-h", "127.0.0.1"}
	if rootPwd != "" {
		args = append(args, "-p"+rootPwd)
	}
	args = append(args, "-e", stmt)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, m.ClientPath(), args...)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("CREATE DATABASE 失败: %v\n%s", err, textenc.ToUTF8(out))
	}
	return nil
}
