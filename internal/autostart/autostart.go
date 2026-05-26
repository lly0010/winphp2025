package autostart

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lly0010/winphp2025/internal/download"
	"github.com/lly0010/winphp2025/internal/extract"
	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/services"
	"github.com/lly0010/winphp2025/internal/sources"
	"github.com/lly0010/winphp2025/internal/wincmd"
)

const PanelTaskName = "WinPHPPanelAutoStart"

// EnsureNssm 确保 nssm.exe 存在; 不存在则下载.
func EnsureNssm(ctx context.Context, prog download.ProgressFn) (string, error) {
	if _, err := os.Stat(paths.NssmFile); err == nil {
		return paths.NssmFile, nil
	}
	src, err := sources.Load()
	if err != nil {
		return "", err
	}
	urls := src.Nssm.AllURLs()
	if len(urls) == 0 {
		return "", fmt.Errorf("sources.json 未配置 nssm")
	}
	zipPath := filepath.Join(paths.TmpDir, "nssm.zip")
	if err := download.DownloadWithRetry(ctx, urls, zipPath, prog, 3); err != nil {
		return "", err
	}
	// 解压, 把 win64/nssm.exe 拷到 bin/
	tmp := filepath.Join(paths.TmpDir, "nssm-extract")
	_ = os.RemoveAll(tmp)
	if err := extract.Zip(zipPath, tmp, ""); err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	defer os.Remove(zipPath)

	// 在解压树里找 win64/nssm.exe
	var found string
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "nssm.exe" && filepath.Base(filepath.Dir(path)) == "win64" {
			found = path
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("解压后未找到 win64/nssm.exe")
	}
	if err := copyFile(found, paths.NssmFile); err != nil {
		return "", err
	}
	logger.Info("NSSM 已安装: %s", paths.NssmFile)
	return paths.NssmFile, nil
}

// SetNssmFromFile 用户手动选择本地 nssm.exe
func SetNssmFromFile(srcPath string) error {
	return copyFile(srcPath, paths.NssmFile)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 64*1024)
	for {
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if rerr != nil {
			break
		}
	}
	return nil
}

// RegisterService 用 NSSM 注册一个 Windows 服务.
func RegisterService(name, exePath string, args []string, workDir, description string, env map[string]string) error {
	if _, err := os.Stat(paths.NssmFile); err != nil {
		return fmt.Errorf("nssm.exe 不存在, 请先 EnsureNssm")
	}
	// 已存在则先移除
	if services.ServiceExists(name) {
		_ = services.StopService(name)
		_ = wincmd.Hidden(paths.NssmFile, "remove", name, "confirm").Run()
	}
	// install
	installArgs := append([]string{"install", name, exePath}, args...)
	if out, err := runNssm(installArgs...); err != nil {
		return fmt.Errorf("nssm install: %v\n%s", err, out)
	}
	_, _ = runNssm("set", name, "AppDirectory", workDir)
	_, _ = runNssm("set", name, "Start", "SERVICE_AUTO_START")
	if description != "" {
		_, _ = runNssm("set", name, "Description", description)
	}
	logDir := filepath.Join(workDir, "logs")
	_ = os.MkdirAll(logDir, 0o755)
	_, _ = runNssm("set", name, "AppStdout", filepath.Join(logDir, "nssm_stdout.log"))
	_, _ = runNssm("set", name, "AppStderr", filepath.Join(logDir, "nssm_stderr.log"))
	if len(env) > 0 {
		var envParts []string
		for k, v := range env {
			envParts = append(envParts, k+"="+v)
		}
		args := append([]string{"set", name, "AppEnvironmentExtra"}, envParts...)
		_, _ = runNssm(args...)
	}
	logger.Info("服务 %s 已注册 (开机自启)", name)
	return nil
}

func UnregisterService(name string) error {
	if !services.ServiceExists(name) {
		return nil
	}
	if _, err := os.Stat(paths.NssmFile); err == nil {
		_, _ = runNssm("stop", name)
		_, _ = runNssm("remove", name, "confirm")
	} else {
		_ = wincmd.Hidden("sc", "stop", name).Run()
		_ = wincmd.Hidden("sc", "delete", name).Run()
	}
	logger.Info("服务 %s 已卸载", name)
	return nil
}

func runNssm(args ...string) (string, error) {
	cmd := wincmd.Hidden(paths.NssmFile, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
