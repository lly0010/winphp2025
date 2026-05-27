// Package portcheck 在 Windows 上诊断端口占用情况.
// nginx 启动失败 (尤其是 10013 / 10048 错误) 时, 我们用它告诉用户:
//   - 端口是否被某个进程绑定 (谁绑的, PID)
//   - 端口是否在 Windows TCP 排除范围里 (WSL2/Hyper-V/Docker 会自动预留一批端口)
// 给出可操作的修复建议.

package portcheck

import "fmt"

type PortInfo struct {
	Port      int    `json:"port"`
	InUse     bool   `json:"inUse"`     // 进程占用 (netstat LISTENING)
	PID       uint32 `json:"pid"`
	ProcName  string `json:"procName"`
	Reserved  bool   `json:"reserved"`  // Windows excluded port range 命中
	Diagnosis string `json:"diagnosis"` // 中文友好诊断字符串 (前端直接展示)
}

// Diagnose 返回端口的诊断信息. 不论端口是否冲突都会返回, InUse / Reserved 字段反映状态.
func Diagnose(port int) PortInfo {
	info := diagnoseImpl(port)
	info.Port = port
	info.Diagnosis = friendly(info)
	return info
}

func friendly(i PortInfo) string {
	if i.InUse {
		name := i.ProcName
		if name == "" {
			name = "(未知进程)"
		}
		return fmt.Sprintf("端口 %d 已被进程 %s (PID %d) 占用. 关闭该进程或换一个端口.", i.Port, name, i.PID)
	}
	if i.Reserved {
		return fmt.Sprintf(
			"端口 %d 没被进程占用, 但被 Windows 系统预留了 (常见于装了 WSL2 / Hyper-V / Docker). "+
				"释放方法: 管理员 CMD 运行 [net stop winnat] -> [net start winnat]. "+
				"或者改用其他端口 (例如把 nginx.conf 里 listen 80 改成 listen 8080).",
			i.Port,
		)
	}
	return fmt.Sprintf("端口 %d 当前空闲 (没占用也没被预留).", i.Port)
}
