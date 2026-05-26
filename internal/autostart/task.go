package autostart

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lly0010/winphp2025/internal/textenc"
	"github.com/lly0010/winphp2025/internal/wincmd"
)

// 面板自启 用 schtasks.exe (而不是 NSSM), 因为面板需要交互登录后启动 (而非系统启动).

var (
	panelCacheMu  sync.Mutex
	panelCacheVal bool
	panelCacheAt  time.Time
)

const panelCacheTTL = 5 * time.Second

// PanelAutoStartEnabled 缓存结果, 避免高频状态轮询频繁调 schtasks.
func PanelAutoStartEnabled() bool {
	panelCacheMu.Lock()
	defer panelCacheMu.Unlock()
	if time.Since(panelCacheAt) < panelCacheTTL {
		return panelCacheVal
	}
	rawOut, err := wincmd.Hidden("schtasks", "/Query", "/TN", PanelTaskName).CombinedOutput()
	out := []byte(textenc.ToUTF8(rawOut))
	panelCacheVal = err == nil && strings.Contains(string(out), PanelTaskName)
	panelCacheAt = time.Now()
	return panelCacheVal
}

// invalidatePanelCache 在 enable/disable 后强制刷新缓存.
func invalidatePanelCache() {
	panelCacheMu.Lock()
	panelCacheAt = time.Time{}
	panelCacheMu.Unlock()
}

func EnablePanelAutoStart(exePath string) error {
	// 先确保任务不存在 (覆盖)
	_ = wincmd.Hidden("schtasks", "/Delete", "/TN", PanelTaskName, "/F").Run()
	// 用最高权限创建, 登录时触发
	cmd := wincmd.Hidden("schtasks",
		"/Create",
		"/TN", PanelTaskName,
		"/TR", "\""+exePath+"\"",
		"/SC", "ONLOGON",
		"/RL", "HIGHEST",
		"/F",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks create: %v\n%s", err, textenc.ToUTF8(out))
	}
	invalidatePanelCache()
	return nil
}

func DisablePanelAutoStart() error {
	rawOut, err := wincmd.Hidden("schtasks", "/Delete", "/TN", PanelTaskName, "/F").CombinedOutput()
	out := textenc.ToUTF8(rawOut)
	if err != nil {
		// 不存在视为成功 (英文 / 中文输出都判)
		if strings.Contains(out, "cannot find") || strings.Contains(out, "does not exist") ||
			strings.Contains(out, "找不到") || strings.Contains(out, "不存在") {
			invalidatePanelCache()
			return nil
		}
		return fmt.Errorf("schtasks delete: %v\n%s", err, out)
	}
	invalidatePanelCache()
	return nil
}

// CurrentExe 返回当前面板可执行文件路径
func CurrentExe() (string, error) {
	return os.Executable()
}
