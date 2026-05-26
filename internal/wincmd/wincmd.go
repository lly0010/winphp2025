// Package wincmd 提供创建无窗口子进程的工具.
// Windows 上所有 exec.Command 默认会闪一个 cmd 控制台, 用这里的 Hidden() 包装解决.

package wincmd

import "os/exec"

// Hidden 等价于 exec.Command, 但子进程不会弹控制台窗口 (Windows).
// 用法: cmd := wincmd.Hidden("schtasks", "/Query", "/TN", "x")
func Hidden(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	hide(cmd)
	return cmd
}

// Apply 给已有的 exec.Cmd 加隐藏窗口属性 (调用方自己 exec.Command 之后再 Apply)
func Apply(cmd *exec.Cmd) { hide(cmd) }
