// Package textenc 处理 Windows 中文系统下子进程输出的 GBK → UTF-8 转换.
// nginx / mysqld / initdb / pg_ctl 等程序在中文 Windows 上往 stderr 写中文时
// 用的是系统 codepage (一般是 GBK/CP936). 我们捕获到 Go 字符串里直接当 UTF-8
// 处理就会显示乱码. 这里检测如果 bytes 不是合法 UTF-8 就按 GBK 解码.

package textenc

import (
	"bytes"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// ToUTF8 智能把 bytes 转成 UTF-8 字符串.
//   - 如果 bytes 已经是合法 UTF-8, 直接返回.
//   - 否则按 GBK (CP936) 解码 (Windows 中文系统默认 codepage).
//   - 再失败就 lossy 返回原 string (用 ? 替换非法字节).
func ToUTF8(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// 剥 UTF-8 BOM (有的话)
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(b) {
		return string(b)
	}
	if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(b); err == nil && utf8.Valid(decoded) {
		return string(decoded)
	}
	// 仍然不行: 把非法字节换成 replacement char
	r := bytes.NewBuffer(nil)
	for len(b) > 0 {
		ru, size := utf8.DecodeRune(b)
		if ru == utf8.RuneError && size == 1 {
			r.WriteRune(0xFFFD)
			b = b[1:]
			continue
		}
		r.WriteRune(ru)
		b = b[size:]
	}
	return r.String()
}
