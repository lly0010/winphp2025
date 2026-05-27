package sources

import (
	"os"
	"path/filepath"
)

// ExpectedBinaries 返回组件安装后应当存在的关键文件 (相对 dest 目录).
// 用户自定义版本/本地 zip 安装后会校验这些文件存在, 否则报"不符合条件".
func ExpectedBinaries(kind string) []string {
	switch kind {
	case "nginx":
		return []string{"nginx.exe"}
	case "php":
		return []string{"php.exe", "php-cgi.exe"}
	case "mysql":
		return []string{"bin/mysqld.exe", "bin/mysql.exe"}
	case "postgresql", "postgres":
		return []string{"bin/postgres.exe", "bin/initdb.exe", "bin/pg_ctl.exe"}
	case "redis":
		return []string{"redis-server.exe", "redis-cli.exe"}
	}
	return nil
}

// VerifyInstall 检查 dest 下是否包含 kind 所需的关键二进制.
// 返回缺失的文件列表 (空则表示验证通过).
func VerifyInstall(kind, dest string) []string {
	var missing []string
	for _, f := range ExpectedBinaries(kind) {
		p := filepath.Join(dest, filepath.FromSlash(f))
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, f)
		}
	}
	return missing
}
