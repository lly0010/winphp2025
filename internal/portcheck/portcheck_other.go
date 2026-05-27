//go:build !windows

package portcheck

func diagnoseImpl(port int) PortInfo {
	return PortInfo{Port: port}
}
