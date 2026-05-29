// Package theme 管理面板主题: 内置主题 + 第三方开发者放在 themes/ 的自定义主题包.
//
// 主题包格式 (放在 themes/<name>/ 目录):
//   themes/my-theme/
//     theme.json     {"id":"my-theme", "name":"我的主题", "author":"xxx", "version":"1.0", "description":"..."}
//     theme.css      标准 CSS, 通常覆盖 :root 里的 CSS 变量 (--primary, --bg 等)
//
// 内置主题 ID:
//   "default"       粉紫二次元 (出厂默认)
//   "blue-classic"  蓝色商务 (原经典风格)
//
// 第三方主题 ID 等于其目录名, 前端显示时用 theme.json 里的 name.

package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lly0010/winphp2025/internal/logger"
	"github.com/lly0010/winphp2025/internal/paths"
)

// Info 主题元数据 (传给前端的简要信息).
type Info struct {
	ID          string `json:"id"`          // 唯一 id: default / blue-classic / 或目录名
	Name        string `json:"name"`        // 显示名 (中文友好)
	Author      string `json:"author,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	Builtin     bool   `json:"builtin"`     // 是否内置
	PreviewBg   string `json:"previewBg,omitempty"` // 预览背景色/渐变 (前端卡片色块)
	PreviewFg   string `json:"previewFg,omitempty"` // 预览前景色
}

// Applied 前端应用主题时拿到的完整内容 (含 CSS).
type Applied struct {
	Info Info   `json:"info"`
	CSS  string `json:"css"` // 自定义主题的 CSS 内容; 内置主题为空 (内置 CSS 在 style.css 里靠 [data-theme] 切换)
}

// 内置主题列表 (前端 ListThemes 会先返回这些).
var builtins = []Info{
	{
		ID: "default", Name: "粉紫二次元", Builtin: true,
		Description: "出厂默认 - 粉紫渐变主题, 萌系风格",
		PreviewBg:   "linear-gradient(135deg, #ff6f9e 0%, #b06fff 100%)",
		PreviewFg:   "#ffffff",
	},
	{
		ID: "blue-classic", Name: "蓝色商务", Builtin: true,
		Description: "原经典蓝色主题, 简洁专业",
		PreviewBg:   "linear-gradient(135deg, #2d74b8 0%, #1e5a92 100%)",
		PreviewFg:   "#ffffff",
	},
	{
		ID: "hibike-euphonium", Name: "吹响吧!上低音号", Builtin: true,
		Description: "京吹同款 - 樱花粉 + 上低音号黄铜色, 加飘落花瓣动画",
		PreviewBg:   "linear-gradient(135deg, #e89bb5 0%, #d4a96a 100%)",
		PreviewFg:   "#ffffff",
	},
}

// List 返回所有可用主题 (内置 + themes/ 目录里的自定义).
func List() []Info {
	out := make([]Info, 0, len(builtins)+4)
	out = append(out, builtins...)
	out = append(out, listCustom()...)
	return out
}

func listCustom() []Info {
	entries, err := os.ReadDir(paths.ThemesDir)
	if err != nil {
		return nil
	}
	var infos []Info
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, ok := readThemeJSON(e.Name())
		if !ok {
			continue
		}
		info.Builtin = false
		// 强制 id = 目录名 (确保唯一)
		info.ID = "custom:" + e.Name()
		infos = append(infos, info)
	}
	return infos
}

func readThemeJSON(dir string) (Info, bool) {
	p := filepath.Join(paths.ThemesDir, dir, "theme.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return Info{}, false
	}
	var info Info
	if err := json.Unmarshal(b, &info); err != nil {
		return Info{}, false
	}
	if info.Name == "" {
		info.Name = dir
	}
	return info, true
}

// Get 返回某个主题的完整信息 (含 CSS, 自定义主题才有).
func Get(id string) (Applied, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "default"
	}
	// 内置
	for _, b := range builtins {
		if b.ID == id {
			return Applied{Info: b, CSS: ""}, nil
		}
	}
	// 自定义: custom:<dir>
	if strings.HasPrefix(id, "custom:") {
		dir := strings.TrimPrefix(id, "custom:")
		info, ok := readThemeJSON(dir)
		if !ok {
			return Applied{}, fmt.Errorf("主题不存在: %s", id)
		}
		info.ID = id
		info.Builtin = false
		cssPath := filepath.Join(paths.ThemesDir, dir, "theme.css")
		css, err := os.ReadFile(cssPath)
		if err != nil {
			return Applied{Info: info}, nil // 没 CSS 也允许
		}
		return Applied{Info: info, CSS: string(css)}, nil
	}
	return Applied{}, fmt.Errorf("未知主题: %s", id)
}

// ValidID 简单校验主题 id 合法 (不能含路径分隔符等).
func ValidID(id string) bool {
	if id == "" {
		return false
	}
	if strings.ContainsAny(id, `/\:`) && !strings.HasPrefix(id, "custom:") {
		return false
	}
	// custom:<name> 中 name 部分不能再含特殊字符
	if strings.HasPrefix(id, "custom:") {
		name := strings.TrimPrefix(id, "custom:")
		if name == "" || strings.ContainsAny(name, `/\:.`) || strings.Contains(name, "..") {
			return false
		}
	}
	return true
}

// EnsureSampleTheme 首次启动时在 themes/ 写一个示例主题包供开发者参考.
func EnsureSampleTheme() {
	dir := filepath.Join(paths.ThemesDir, "example")
	if _, err := os.Stat(dir); err == nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "theme.json"), []byte(sampleJSON), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "theme.css"), []byte(sampleCSS), 0o644)
	_ = os.WriteFile(filepath.Join(paths.ThemesDir, "README.txt"), []byte(devReadme), 0o644)
	logger.Info("已写入示例主题: themes/example/")
}

// RemoveCustom 删除一个自定义主题目录 (内置不可删).
func RemoveCustom(id string) error {
	if !strings.HasPrefix(id, "custom:") {
		return errors.New("只能删除自定义主题")
	}
	dir := strings.TrimPrefix(id, "custom:")
	if dir == "" || strings.ContainsAny(dir, `/\:.`) {
		return errors.New("非法主题 id")
	}
	target := filepath.Join(paths.ThemesDir, dir)
	return os.RemoveAll(target)
}

const sampleJSON = `{
  "name": "示例主题 (深空)",
  "author": "your-name",
  "version": "1.0.0",
  "description": "这是给开发者参考的示例主题. 复制 themes/example/ 目录改名即可创建自己的主题包."
}
`

const sampleCSS = `/* 示例主题 - 深空蓝
 * 主题作者可在这里覆盖 :root 里的 CSS 变量来改外观.
 * 完整变量清单见 frontend/src/style.css.
 *
 * 注意: 这段 CSS 会在用户切换到本主题时通过 <style id="theme-custom">
 * 注入到 <head>, 优先级高于内置主题.
 */

:root {
  --primary:       #4a8edd;
  --primary-dark:  #2d6cbe;
  --primary-light: #e1ecfa;
  --accent:        #6fbfff;
  --accent-light:  #e3f2ff;

  --bg:            #0f1a2e;
  --bg-card:       rgba(28, 41, 64, 0.92);
  --border:        #2a3a55;
  --border-soft:   rgba(111, 191, 255, 0.20);

  --text:          #e6ecf5;
  --text-secondary:#9caac1;
  --text-disabled: #5b6878;

  --shadow:       0 2px 8px rgba(0,0,0,0.30), 0 6px 20px rgba(74,142,221,0.18);
  --shadow-hover: 0 4px 14px rgba(0,0,0,0.40), 0 10px 32px rgba(74,142,221,0.30);

  --header-grad:  linear-gradient(135deg, #4a8edd 0%, #6fbfff 100%);
  --sidebar-bg:   linear-gradient(180deg, #050d1e 0%, #0f1f3a 100%);
}

/* 暗色主题下的细节调整 */
input, select, textarea {
  background: rgba(255,255,255,0.06);
  color: var(--text);
  border-color: var(--border);
}
.modal { background: #1c2940; color: var(--text); }
.modal-footer { background: #131e33; }
.table th { background: linear-gradient(90deg, rgba(74,142,221,0.10), rgba(111,191,255,0.10)); }
`

const devReadme = `WinPHP 主题包开发说明
=============================

这个 themes/ 目录是给第三方开发者放主题包用的.

每个主题是一个子目录, 含两个文件:
  themes/<your-theme-id>/
    theme.json    主题元数据
    theme.css     CSS, 通常覆盖 :root 里的变量

theme.json 字段:
  - name         (必填) 显示名, 例如 "深空"
  - author       作者
  - version      版本号
  - description  简介

theme.css 怎么写:
  覆盖 frontend/src/style.css 里 :root 的 CSS 变量.
  常用变量:
    --primary / --primary-dark / --primary-light
    --accent / --accent-light
    --bg / --bg-card / --border / --border-soft
    --text / --text-secondary / --text-disabled
    --shadow / --shadow-hover
    --header-grad / --sidebar-bg
  详细看 themes/example/theme.css.

写好后:
  1. 打开 WinPHP 面板
  2. 工具页 → "🎨 切换主题"
  3. 在对话框里选你的主题应用

也可以做主题包压缩分发:
  把 themes/your-theme/ 整个文件夹压缩成 zip,
  用户解压到自己的 themes/ 目录就能用了.
`
