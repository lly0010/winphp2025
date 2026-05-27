//go:build windows

package winshort

import (
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetShortPath = kernel32.NewProc("GetShortPathNameW")
)

func shortImpl(path string) string {
	if path == "" {
		return path
	}
	pw, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return path
	}
	// 先用栈上 buffer 试
	var buf [1024]uint16
	ret, _, _ := procGetShortPath.Call(
		uintptr(unsafe.Pointer(pw)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		// 转换失败 (路径不存在 / NTFS 关了 8.3 / 其他错误)
		return path
	}
	if int(ret) <= len(buf) {
		return syscall.UTF16ToString(buf[:ret])
	}
	// buffer 不够大: 用返回的长度重试
	big := make([]uint16, ret+1)
	ret2, _, _ := procGetShortPath.Call(
		uintptr(unsafe.Pointer(pw)),
		uintptr(unsafe.Pointer(&big[0])),
		uintptr(len(big)),
	)
	if ret2 == 0 {
		return path
	}
	return syscall.UTF16ToString(big[:ret2])
}
