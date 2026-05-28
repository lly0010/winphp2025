// Package winshort 提供获取 Windows 8.3 短路径的工具.
// nginx / mysqld / 一些 C 程序在 Windows 上对含中文 (非 ASCII) 的路径处理有问题:
// nginx 启动时会因为 fopen 用 ANSI codepage 而把 unicode 路径转码失败.
// 用 Win32 API GetShortPathNameW 把长路径转成 C:\6CBEAA~1\... 这种纯 ASCII 短路径就能正常工作.
//
// 前提: NTFS 卷启用了 8.3 短文件名生成 (默认开启). 部分 SSD 优化指南建议关掉它,
// 那样 Short() 会返回原路径, 中文路径下还是有可能启动失败.

package winshort

// Short 返回 path 的 Windows 8.3 短路径. 路径必须已存在.
// 非 Windows 平台或转换失败时返回原 path.
func Short(path string) string {
	return shortImpl(path)
}

// ShortIfNeeded 仅当路径含非 ASCII 字符时才转短路径; 全 ASCII 直接原样返回.
// 用这个能避免给英文路径无端转出 'DOWNLO~1' 这种短名导致 nginx 等程序
// 拼接路径时产生混合斜杠 bug.
func ShortIfNeeded(path string) string {
	for _, r := range path {
		if r > 127 {
			return shortImpl(path)
		}
	}
	return path
}
