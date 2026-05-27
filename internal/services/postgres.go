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
	"github.com/lly0010/winphp2025/internal/textenc"
)

const PostgresServiceName = "WinPHPPostgreSQL"

type Postgres struct{}

func (Postgres) Name() string         { return "postgresql" }
func (Postgres) ExePath() string      { return filepath.Join(paths.PgDir, "bin", "postgres.exe") }
func (Postgres) InitdbPath() string   { return filepath.Join(paths.PgDir, "bin", "initdb.exe") }
func (Postgres) PgCtlPath() string    { return filepath.Join(paths.PgDir, "bin", "pg_ctl.exe") }
func (Postgres) PsqlPath() string     { return filepath.Join(paths.PgDir, "bin", "psql.exe") }
func (Postgres) DataPath() string     { return filepath.Join(paths.PgDir, "data") }
func (Postgres) LogPath() string      { return filepath.Join(paths.PgDir, "logs", "postgres.log") }

func (p Postgres) Version() string {
	if _, err := os.Stat(p.ExePath()); err != nil {
		return ""
	}
	out, err := runHidden(p.ExePath(), 3*time.Second, "--version")
	if err != nil {
		return ""
	}
	// "postgres (PostgreSQL) 17.2"
	parts := strings.Fields(out)
	for _, w := range parts {
		if len(w) > 0 && (w[0] >= '0' && w[0] <= '9') {
			return strings.TrimSpace(w)
		}
	}
	return ""
}

func (p Postgres) Status() Status {
	binOk := false
	if _, err := os.Stat(p.ExePath()); err == nil {
		binOk = true
	}
	svcInstalled := ServiceExists(PostgresServiceName)
	svc := ""
	if svcInstalled {
		svc = ServiceStatusStr(PostgresServiceName)
	}
	running := svc == "Running"
	if !running && binOk {
		running = proc.HasProcessByPathPrefix("postgres", paths.PgDir)
	}
	if !running && binOk {
		// pg_ctl 用受限令牌 spawn postgres, MainModule 可能读不到, 端口兜底
		running = proc.PortListening(5432)
	}
	return Status{
		Running: running, Port: 5432, Version: p.Version(),
		ServiceInstalled: svcInstalled, ServiceStatus: svc, BinInstalled: binOk,
	}
}

func (p Postgres) initData() error {
	dataDir := p.DataPath()
	if _, err := os.Stat(filepath.Join(dataDir, "PG_VERSION")); err == nil {
		return nil
	}
	_ = os.RemoveAll(dataDir)
	if _, err := os.Stat(p.InitdbPath()); err != nil {
		return fmt.Errorf("initdb.exe 不存在")
	}
	_ = os.MkdirAll(filepath.Join(paths.PgDir, "logs"), 0o755)
	logger.Info("PostgreSQL 正在初始化 data 目录...")
	out, err := runHidden(p.InitdbPath(), 3*time.Minute,
		"-D", dataDir, "-E", "UTF8", "--locale=C", "-U", "postgres", "--auth=trust")
	if err != nil {
		return fmt.Errorf("initdb: %v\n%s", err, out)
	}
	// 写入我们的 pg_hba.conf 和追加 postgresql.conf
	hba, _ := readTemplate("pg_hba.conf", defaultPgHba)
	_ = os.WriteFile(filepath.Join(dataDir, "pg_hba.conf"), []byte(hba), 0o644)

	cfg, _ := readTemplate("postgresql.conf", defaultPgConf)
	cfgPath := filepath.Join(dataDir, "postgresql.conf")
	if b, rerr := os.ReadFile(cfgPath); rerr == nil {
		merged := string(b) + "\n# ===== Appended by WinPHP =====\n" + cfg + "\n"
		_ = os.WriteFile(cfgPath, []byte(merged), 0o644)
	}

	st := state.Load()
	st.PgInited = true
	_ = state.Save(st)
	logger.Info("PostgreSQL 初始化完成, superuser=postgres, 本地 trust 无密码")
	return nil
}

func (p Postgres) Start() error {
	if _, err := os.Stat(p.PgCtlPath()); err != nil {
		return fmt.Errorf("PostgreSQL 未安装")
	}
	if p.Status().Running {
		return fmt.Errorf("PostgreSQL 已运行")
	}
	if proc.PortListening(5432) {
		return fmt.Errorf("端口 5432 已被占用. %s", portcheck.Diagnose(5432).Diagnosis)
	}
	if err := p.initData(); err != nil {
		return err
	}
	if ServiceExists(PostgresServiceName) {
		return StartService(PostgresServiceName)
	}
	cmd := exec.Command(p.PgCtlPath(), "start", "-D", p.DataPath(), "-l", p.LogPath(), "-w", "-t", "30")
	cmd.Dir = paths.PgDir
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_ctl start: %v\n%s", err, textenc.ToUTF8(out))
	}
	time.Sleep(500 * time.Millisecond)
	logger.Info("PostgreSQL 启动")
	return nil
}

func (p Postgres) Stop() error {
	if ServiceExists(PostgresServiceName) {
		_ = StopService(PostgresServiceName)
	}
	if _, err := os.Stat(p.PgCtlPath()); err == nil {
		if _, err := os.Stat(p.DataPath()); err == nil {
			cmd := exec.Command(p.PgCtlPath(), "stop", "-D", p.DataPath(), "-m", "fast", "-w", "-t", "30")
			cmd.Dir = paths.PgDir
			hideWindow(cmd)
			_ = cmd.Run()
		}
	}
	time.Sleep(500 * time.Millisecond)
	killByPathPrefix("postgres", paths.PgDir)
	logger.Info("PostgreSQL 已停止")
	return nil
}

func (p Postgres) Restart() error {
	_ = p.Stop()
	time.Sleep(500 * time.Millisecond)
	return p.Start()
}

const defaultPgConf = `listen_addresses = 'localhost'
port = 5432
max_connections = 100
shared_buffers = 128MB
dynamic_shared_memory_type = windows
log_destination = 'stderr'
logging_collector = on
log_directory = 'log'
log_filename = 'postgresql.log'
log_rotation_size = 10MB
log_min_messages = warning
datestyle = 'iso, mdy'
timezone = 'Asia/Shanghai'
default_text_search_config = 'pg_catalog.simple'
`

const defaultPgHba = `# WinPHP PostgreSQL 认证 (开发模式: 本地 trust 无密码)
local   all all                  trust
host    all all 127.0.0.1/32     trust
host    all all ::1/128          trust
`

func (Postgres) InitConfig() error {
	if _, err := os.Stat(paths.PgDir); err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Join(paths.PgDir, "logs"), 0o755)
	return nil
}
