package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lly0010/winphp2025/internal/download"
	"github.com/lly0010/winphp2025/internal/extract"
	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
)

// InstallableExt 一个可在线安装的 PHP 扩展 (从 PECL 拉取预编译 DLL).
type InstallableExt struct {
	Name      string   `json:"name"`      // 短名: "redis" -> 装出 php_redis.dll, php.ini 写 extension=redis
	Display   string   `json:"display"`   // 显示名
	Type      string   `json:"type"`      // "extension" / "zend_extension"
	Versions  []string `json:"versions"`  // 可选版本号 (PECL 上的)
	Default   string   `json:"default"`   // 默认选中版本
	Deps      []string `json:"deps"`      // 依赖的其他扩展 (会一起装)
	Note      string   `json:"note,omitempty"`
}

// KnownInstallableExts 内置的"一键装"扩展清单. 用户也可以编辑 php.ini 自己改.
func KnownInstallableExts() []InstallableExt {
	return []InstallableExt{
		{
			Name: "redis", Display: "Redis 客户端", Type: "extension",
			Versions: []string{"6.1.0", "6.0.2", "5.3.7"}, Default: "6.0.2",
			Deps: []string{"igbinary"},
			Note: "连接 Redis 服务. 通常配合 igbinary 序列化使用.",
		},
		{
			Name: "igbinary", Display: "igbinary (二进制序列化, redis 依赖)", Type: "extension",
			Versions: []string{"3.2.16", "3.2.15"}, Default: "3.2.16",
			Note: "Redis / Memcached 的高效序列化后端.",
		},
		{
			Name: "memcached", Display: "Memcached 客户端", Type: "extension",
			Versions: []string{"3.3.0"}, Default: "3.3.0",
			Deps: []string{"igbinary"},
		},
		{
			Name: "mongodb", Display: "MongoDB 客户端", Type: "extension",
			Versions: []string{"1.21.0", "1.20.1"}, Default: "1.21.0",
		},
		{
			Name: "xdebug", Display: "Xdebug 调试器", Type: "zend_extension",
			Versions: []string{"3.4.0", "3.3.2"}, Default: "3.4.0",
			Note: "PHP 调试 + Profile. 启用后会拖慢, 仅开发用.",
		},
		{
			Name: "imagick", Display: "Imagick (图像处理)", Type: "extension",
			Versions: []string{"3.7.0"}, Default: "3.7.0",
			Note: "需要先装 ImageMagick 主程序到系统 PATH.",
		},
	}
}

// detectPhpInfo 跑 php -v 解析版本和 VS 编译标签.
func (p PHP) detectPhpInfo() (phpVer, vsTag string, err error) {
	out, err := runHidden(p.ExePath(), 3*time.Second, "-v")
	if err != nil {
		return "", "", fmt.Errorf("php -v 失败: %w", err)
	}
	// 第一行: "PHP 8.3.14 (cli) (built: ...) (NTS Visual C++ 2019 x64)"
	re := regexp.MustCompile(`PHP\s+(\d+\.\d+(?:\.\d+)?)`)
	m := re.FindStringSubmatch(out)
	if m == nil {
		return "", "", fmt.Errorf("无法解析 PHP 版本: %s", strings.SplitN(out, "\n", 2)[0])
	}
	phpVer = m[1]
	// VS 标签: PHP 8.4+ → vs17; 8.0-8.3 → vs16; 7.x → vc15
	parts := strings.SplitN(phpVer, ".", 3)
	mm := phpVer
	if len(parts) >= 2 {
		mm = parts[0] + "." + parts[1]
	}
	switch {
	case strings.HasPrefix(mm, "8.4") || strings.HasPrefix(mm, "8.5"):
		vsTag = "vs17"
	case strings.HasPrefix(mm, "8."):
		vsTag = "vs16"
	case strings.HasPrefix(mm, "7."):
		vsTag = "vc15"
	default:
		vsTag = "vs16"
	}
	return phpVer, vsTag, nil
}

// peclURL 按 PECL 命名规则构造 Windows zip 下载 URL.
//   https://windows.php.net/downloads/pecl/releases/<name>/<ver>/
//     php_<name>-<ver>-<phpMM>-nts-<vsTag>-x64.zip
func peclURL(name, extVer, phpMM, vsTag string, ts bool) string {
	tsTag := "nts"
	if ts {
		tsTag = "ts"
	}
	return fmt.Sprintf(
		"https://windows.php.net/downloads/pecl/releases/%s/%s/php_%s-%s-%s-%s-%s-x64.zip",
		name, extVer, name, extVer, phpMM, tsTag, vsTag,
	)
}

// findInstallableExt 在内置清单里按名字找元数据.
func findInstallableExt(name string) *InstallableExt {
	for _, e := range KnownInstallableExts() {
		if e.Name == name {
			ext := e
			return &ext
		}
	}
	return nil
}

// hasExtDLL 看 ext/ 下是否已有该扩展的 dll (php_<name>.dll).
// 注意: 一个扩展可能解压出多个 dll, 我们以主 dll 名字判断.
func (p PHP) hasExtDLL(name string) bool {
	candidates := []string{
		filepath.Join(paths.PhpDir, "ext", "php_"+name+".dll"),
		filepath.Join(paths.PhpDir, "ext", name+".dll"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return true
		}
	}
	return false
}

// InstallExtensionFromPECL 从 PECL 下载并安装一个 PHP 扩展.
// 自动递归装依赖 (Deps, 用各依赖的默认版本; 已装的跳过).
// 流程: 检测 php 版本 → 装依赖 → 构造 URL → 下载 zip → 解压 *.dll → 改 php.ini.
// 重启 PHP-CGI 才生效.
func (p PHP) InstallExtensionFromPECL(ctx context.Context, name, extVer string, prog download.ProgressFn) error {
	return p.installExtWithDeps(ctx, name, extVer, prog, map[string]bool{})
}

// installExtWithDeps 递归装依赖再装自己. visited 防循环.
func (p PHP) installExtWithDeps(ctx context.Context, name, extVer string, prog download.ProgressFn, visited map[string]bool) error {
	if visited[name] {
		return nil
	}
	visited[name] = true

	// 先装依赖 (按内置清单的 default 版本)
	if info := findInstallableExt(name); info != nil {
		for _, dep := range info.Deps {
			if p.hasExtDLL(dep) {
				logger.Info("依赖 %s 已安装, 跳过", dep)
				continue
			}
			depInfo := findInstallableExt(dep)
			if depInfo == nil {
				logger.Warn("依赖 %s 不在内置扩展清单, 无法自动装. 主扩展可能会因缺失依赖加载失败", dep)
				continue
			}
			depVer := depInfo.Default
			if depVer == "" && len(depInfo.Versions) > 0 {
				depVer = depInfo.Versions[0]
			}
			logger.Info("自动安装依赖: %s %s", dep, depVer)
			if err := p.installExtWithDeps(ctx, dep, depVer, prog, visited); err != nil {
				// 依赖装失败只警告, 继续装主扩展. 主扩展可能不需要依赖也能用.
				logger.Warn("依赖 %s 安装失败 (继续装主扩展): %v", dep, err)
			}
		}
	}

	return p.installOneExt(ctx, name, extVer, prog)
}

// installOneExt 装单个扩展 (不递归依赖).
func (p PHP) installOneExt(ctx context.Context, name, extVer string, prog download.ProgressFn) error {
	if _, err := os.Stat(p.ExePath()); err != nil {
		return fmt.Errorf("PHP 未安装, 请先到首页安装 PHP")
	}
	phpVer, vsTag, err := p.detectPhpInfo()
	if err != nil {
		return err
	}
	parts := strings.SplitN(phpVer, ".", 3)
	phpMM := phpVer
	if len(parts) >= 2 {
		phpMM = parts[0] + "." + parts[1]
	}

	// 下载: 先试 nts, 失败试 ts (用户装的可能是 ts)
	tmpZip := filepath.Join(paths.TmpDir, "php_ext_"+name+"_"+extVer+".zip")
	urls := []string{
		peclURL(name, extVer, phpMM, vsTag, false),
		peclURL(name, extVer, phpMM, vsTag, true),
	}
	if err := download.DownloadWithRetry(ctx, urls, tmpZip, prog, 2); err != nil {
		return fmt.Errorf("下载扩展 %s 失败: %w\n常见原因: 该版本对你的 PHP %s (%s) 不可用, 换个版本试试", name, err, phpVer, vsTag)
	}
	defer os.Remove(tmpZip)

	// 解压到临时目录, 然后只挑 *.dll 拷到 ext/
	extractDir := filepath.Join(paths.TmpDir, "php_ext_extract_"+name)
	_ = os.RemoveAll(extractDir)
	if err := extract.Zip(tmpZip, extractDir, ""); err != nil {
		return fmt.Errorf("解压 %s 失败: %w", name, err)
	}
	defer os.RemoveAll(extractDir)

	extDir := filepath.Join(paths.PhpDir, "ext")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		return err
	}
	var copied []string
	_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		n := info.Name()
		if !strings.HasSuffix(strings.ToLower(n), ".dll") {
			return nil
		}
		dst := filepath.Join(extDir, n)
		if err := copyOne(path, dst); err != nil {
			logger.Warn("拷贝 %s 失败: %v", n, err)
			return nil
		}
		copied = append(copied, n)
		return nil
	})
	if len(copied) == 0 {
		return fmt.Errorf("%s zip 内没找到 *.dll, 可能包结构异常", name)
	}
	logger.Info("PHP 扩展 %s %s 已装: %v", name, extVer, copied)

	// 改 php.ini 启用主扩展
	if err := p.SetExtension(name, true); err != nil {
		return fmt.Errorf("写 php.ini 失败 (%s): %w", name, err)
	}
	return nil
}

func copyOne(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// InstallExtensionFromURL 用用户提供的自定义 URL 下载安装一个 PHP 扩展.
// urls: 一个或多个候选下载地址 (依次重试). 接受 zip (会解压挑 dll) 或单个 .dll 直链.
// name: 扩展短名 (用来写 php.ini 的 extension=<name>); 空就从文件名推.
// 不自动装依赖 — 自定义场景假定用户自己清楚.
func (p PHP) InstallExtensionFromURL(ctx context.Context, name string, urls []string, prog download.ProgressFn) error {
	if _, err := os.Stat(p.ExePath()); err != nil {
		return fmt.Errorf("PHP 未安装, 请先到首页安装 PHP")
	}
	if len(urls) == 0 {
		return fmt.Errorf("请至少填一个下载 URL")
	}
	firstURL := urls[0]
	lowerURL := strings.ToLower(firstURL)
	isDll := strings.HasSuffix(lowerURL, ".dll")
	isZip := strings.HasSuffix(lowerURL, ".zip") || !isDll

	tag := name
	if tag == "" {
		tag = "custom"
	}
	tmpPath := filepath.Join(paths.TmpDir, "php_ext_url_"+tag)
	if isZip {
		tmpPath += ".zip"
	} else {
		tmpPath += ".dll"
	}
	if err := download.DownloadWithRetry(ctx, urls, tmpPath, prog, 2); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(tmpPath)

	extDir := filepath.Join(paths.PhpDir, "ext")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		return err
	}

	var copied []string
	if isDll {
		dstName := filepath.Base(firstURL)
		if !strings.HasPrefix(strings.ToLower(dstName), "php_") {
			dstName = "php_" + dstName
		}
		if err := copyOne(tmpPath, filepath.Join(extDir, dstName)); err != nil {
			return fmt.Errorf("拷贝 dll 失败: %w", err)
		}
		copied = append(copied, dstName)
		if name == "" {
			n := strings.ToLower(strings.TrimSuffix(dstName, ".dll"))
			name = strings.TrimPrefix(n, "php_")
		}
	} else {
		extractDir := filepath.Join(paths.TmpDir, "php_ext_url_extract_"+tag)
		_ = os.RemoveAll(extractDir)
		if err := extract.Zip(tmpPath, extractDir, ""); err != nil {
			return fmt.Errorf("解压失败: %w", err)
		}
		defer os.RemoveAll(extractDir)
		_ = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			n := info.Name()
			if !strings.HasSuffix(strings.ToLower(n), ".dll") {
				return nil
			}
			dst := filepath.Join(extDir, n)
			if err := copyOne(path, dst); err != nil {
				logger.Warn("拷贝 %s 失败: %v", n, err)
				return nil
			}
			copied = append(copied, n)
			return nil
		})
		if len(copied) == 0 {
			return fmt.Errorf("zip 内没找到 *.dll, 可能包结构异常或不是扩展包")
		}
		if name == "" {
			for _, n := range copied {
				lo := strings.ToLower(n)
				if strings.HasPrefix(lo, "php_") {
					name = strings.TrimSuffix(strings.TrimPrefix(lo, "php_"), ".dll")
					break
				}
			}
		}
	}
	if name == "" {
		return fmt.Errorf("装好了 dll 但推断不出扩展名, 请在表单里手动填 name")
	}
	logger.Info("PHP 扩展 %s 已装 (自定义 URL): %v", name, copied)
	if err := p.SetExtension(name, true); err != nil {
		return fmt.Errorf("写 php.ini 失败 (%s): %w", name, err)
	}
	return nil
}

// UninstallExtension 卸载扩展: 删 ext/ 下 php_<name>.dll + 注释掉 php.ini 里的 extension=<name>.
// PHP-CGI 重启后完全生效. 删除 dll 失败 (CGI 持有句柄) 时仅日志告警.
func (p PHP) UninstallExtension(name string) error {
	if name == "" {
		return fmt.Errorf("扩展名为空")
	}
	if strings.ContainsAny(name, `/\:.`) {
		return fmt.Errorf("非法扩展名")
	}
	if err := p.SetExtension(name, false); err != nil {
		return fmt.Errorf("修改 php.ini 失败: %w", err)
	}
	candidates := []string{
		filepath.Join(paths.PhpDir, "ext", "php_"+name+".dll"),
		filepath.Join(paths.PhpDir, "ext", name+".dll"),
	}
	removed := []string{}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			if err := os.Remove(c); err != nil {
				logger.Warn("删除 %s 失败: %v (PHP-CGI 可能持有, 重启后再试)", c, err)
			} else {
				removed = append(removed, filepath.Base(c))
			}
		}
	}
	logger.Info("PHP 扩展 %s 已卸载. 删除文件: %v (重启 PHP-CGI 完全生效)", name, removed)
	return nil
}
