//go:build windows

package proc

import (
	"strings"
	"syscall"
	"unsafe"
)

// 使用 ToolHelp32 快速枚举进程. 比起 PowerShell Get-Process 几十倍快.

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snap   = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW        = kernel32.NewProc("Process32FirstW")
	procProcess32NextW         = kernel32.NewProc("Process32NextW")
	procCloseHandle            = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageW = kernel32.NewProc("QueryFullProcessImageNameW")
	procOpenProcess            = kernel32.NewProc("OpenProcess")
)

const (
	TH32CS_SNAPPROCESS              = 0x00000002
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	INVALID_HANDLE_VALUE            = ^uintptr(0)
)

type processEntry32W struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

func hasProcessImpl(nameLower, pathPrefixLower string) bool {
	snap, _, _ := procCreateToolhelp32Snap.Call(TH32CS_SNAPPROCESS, 0)
	if snap == INVALID_HANDLE_VALUE {
		return false
	}
	defer procCloseHandle.Call(snap)

	var entry processEntry32W
	entry.dwSize = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return false
	}
	for {
		exe := syscall.UTF16ToString(entry.szExeFile[:])
		exeL := strings.ToLower(exe)
		// 1) 名字匹配
		if nameLower != "" {
			matched := exeL == nameLower+".exe" || exeL == nameLower
			if matched {
				if pathPrefixLower == "" {
					return true
				}
				// 2) 查询完整路径
				full, ok := queryFullImagePath(entry.th32ProcessID)
				if ok && strings.HasPrefix(strings.ToLower(full), pathPrefixLower) {
					return true
				}
			}
		}
		ret, _, _ := procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}
	return false
}

func queryFullImagePath(pid uint32) (string, bool) {
	h, _, _ := procOpenProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if h == 0 {
		return "", false
	}
	defer procCloseHandle.Call(h)
	var buf [1024]uint16
	size := uint32(len(buf))
	ret, _, _ := procQueryFullProcessImageW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf[:size]), true
}
