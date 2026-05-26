package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// 面板自启 用 schtasks.exe (而不是 NSSM), 因为面板需要交互登录后启动 (而非系统启动).

func PanelAutoStartEnabled() bool {
	out, err := exec.Command("schtasks", "/Query", "/TN", PanelTaskName).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), PanelTaskName)
}

func EnablePanelAutoStart(exePath string) error {
	// 先确保任务不存在 (覆盖)
	_ = exec.Command("schtasks", "/Delete", "/TN", PanelTaskName, "/F").Run()
	// 用最高权限创建, 登录时触发
	args := []string{
		"/Create",
		"/TN", PanelTaskName,
		"/TR", "\"" + exePath + "\"",
		"/SC", "ONLOGON",
		"/RL", "HIGHEST",
		"/F",
	}
	cmd := exec.Command("schtasks", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks create: %v\n%s", err, out)
	}
	return nil
}

func DisablePanelAutoStart() error {
	out, err := exec.Command("schtasks", "/Delete", "/TN", PanelTaskName, "/F").CombinedOutput()
	if err != nil {
		// 不存在视为成功
		if strings.Contains(string(out), "cannot find") || strings.Contains(string(out), "does not exist") {
			return nil
		}
		return fmt.Errorf("schtasks delete: %v\n%s", err, out)
	}
	return nil
}

// CurrentExe 返回当前面板可执行文件路径
func CurrentExe() (string, error) {
	return os.Executable()
}
