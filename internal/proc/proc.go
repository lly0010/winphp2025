package proc

import (
	"net"
	"strings"
	"time"
)

// 进程检测: 用 Windows ToolHelp32 通过 syscall, 比 PowerShell 的 Get-Process 快几十倍.
// 同时利用端口监听检测作为兜底 (postgres 用受限令牌运行时, MainModule 不可读).

// PortListening 检测 127.0.0.1:port 是否在监听.
func PortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+itoa(port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [16]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

// HasProcessByPathPrefix 检测是否存在 image path 以 pathPrefix 开头的运行中进程.
// 实现在 proc_windows.go (build tag windows).
func HasProcessByPathPrefix(name, pathPrefix string) bool {
	return hasProcessImpl(strings.ToLower(name), strings.ToLower(pathPrefix))
}
