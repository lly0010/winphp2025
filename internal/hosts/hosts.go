package hosts

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lly0010/winphp2025/internal/paths"
)

const tag = "# WinPHP"

func Add(domain string) error {
	if domain == "" || domain == "localhost" {
		return nil
	}
	if exists(domain) {
		return nil
	}
	f, err := os.OpenFile(paths.HostsFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("无法打开 hosts (需管理员权限): %w", err)
	}
	defer f.Close()
	line := fmt.Sprintf("\n127.0.0.1\t%s\t%s\n", domain, tag)
	_, err = f.WriteString(line)
	return err
}

func Remove(domain string) error {
	b, err := os.ReadFile(paths.HostsFile)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`(?m)^\s*[\d\.]+\s+` + regexp.QuoteMeta(domain) + `(\s|$).*` + regexp.QuoteMeta(tag) + `.*$\n?`)
	out := pattern.ReplaceAll(b, nil)
	return os.WriteFile(paths.HostsFile, out, 0o644)
}

func exists(domain string) bool {
	f, err := os.Open(paths.HostsFile)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	pattern := regexp.MustCompile(`^\s*[\d\.]+\s+` + regexp.QuoteMeta(domain) + `(\s|$)`)
	for scanner.Scan() {
		if pattern.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}

// Read 返回 hosts 内容 (前端编辑用)
func Read() (string, error) {
	b, err := os.ReadFile(paths.HostsFile)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})), nil
}

func Write(content string) error {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(paths.HostsFile, []byte(content), 0o644)
}
