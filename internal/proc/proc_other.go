//go:build !windows

package proc

// 在非 Windows 平台编译时的占位 (用于在 Linux/Mac 上 go build -tags 跑测试).
func hasProcessImpl(nameLower, pathPrefixLower string) bool {
	return false
}
