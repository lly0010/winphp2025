//go:build windows

package portcheck

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/lly0010/winphp2025/internal/textenc"
	"github.com/lly0010/winphp2025/internal/wincmd"
)

func diagnoseImpl(port int) PortInfo {
	info := PortInfo{Port: port}

	// 1) 查 netstat -ano 看是否有进程 LISTENING 该端口
	if pid, ok := findListenerPID(port); ok {
		info.InUse = true
		info.PID = pid
		info.ProcName = pidName(pid)
	}

	// 2) 查 Windows TCP excluded port range (HNS / WSL / Docker 预留)
	info.Reserved = isReserved(port)
	return info
}

// findListenerPID 用 netstat -ano 找哪个 PID 在 LISTENING 该端口.
// 返回值: pid, found
func findListenerPID(port int) (uint32, bool) {
	out, _ := wincmd.Hidden("netstat", "-ano", "-p", "TCP").Output()
	text := textenc.ToUTF8(out)
	// 匹配类似: "  TCP    0.0.0.0:80     0.0.0.0:0    LISTENING    1234"
	re := regexp.MustCompile(`(?m)^\s*TCP\s+\S*:` + strconv.Itoa(port) + `\s+\S+\s+LISTENING\s+(\d+)\s*$`)
	m := re.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	pid64, err := strconv.ParseUint(m[1], 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(pid64), true
}

// pidName 用 tasklist 查 PID 对应的 EXE 名.
func pidName(pid uint32) string {
	out, _ := wincmd.Hidden("tasklist", "/fi", "PID eq "+strconv.FormatUint(uint64(pid), 10), "/fo", "csv", "/nh").Output()
	text := textenc.ToUTF8(out)
	// "name.exe","12345","Console","1","2,345 K"
	re := regexp.MustCompile(`^"([^"]+)"`)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := re.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// isReserved 调 netsh 看端口是否在 TCP 排除范围 (Windows reserved).
func isReserved(port int) bool {
	out, _ := wincmd.Hidden("netsh", "int", "ipv4", "show", "excludedportrange", "protocol=tcp").Output()
	text := textenc.ToUTF8(out)
	// 输出格式 (英文 Windows):
	//   Start Port    End Port
	//   ----------    --------
	//        50000       50059
	// 中文 Windows 是 "启动端口  结束端口"
	re := regexp.MustCompile(`(?m)^\s*(\d+)\s+(\d+)\s*$`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		startPort, _ := strconv.Atoi(m[1])
		endPort, _ := strconv.Atoi(m[2])
		if port >= startPort && port <= endPort {
			return true
		}
	}
	return false
}
