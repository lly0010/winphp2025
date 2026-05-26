package services

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/lly0010/winphp2025/internal/textenc"
)

// runHidden 执行命令, 隐藏窗口, 合并 stdout+stderr, 超时杀掉.
// 子进程在中文 Windows 上写 stderr 用 GBK codepage, textenc.ToUTF8 自动检测转码.
func runHidden(name string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	hideWindow(cmd)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return textenc.ToUTF8(buf.Bytes()), err
}

// killByPathPrefix 杀掉 image path 以 prefix 开头的进程.
// Windows 实现走 taskkill 兜底 (proc 模块只检测不杀).
func killByPathPrefix(name, prefix string) {
	// 使用 taskkill 按 image name 杀, 然后由 caller 确认是否还残留
	cmd := exec.Command("taskkill", "/F", "/IM", name+".exe")
	hideWindow(cmd)
	_ = cmd.Run()
}

// readTemplate 优先读 config/templates/<name>, 不存在用 fallback.
func readTemplate(name, fallback string) (string, error) {
	p := filepath.Join(paths.TplDir, name)
	if b, err := os.ReadFile(p); err == nil {
		return string(b), nil
	}
	return fallback, nil
}

// readFileAll 读全文 (用于配置编辑器)
func readFileAll(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	s := string(b)
	// 剥 UTF-8 BOM (EF BB BF)
	if len(s) >= 3 && s[0] == 0xEF && s[1] == 0xBB && s[2] == 0xBF {
		s = s[3:]
	}
	return s, nil
}
