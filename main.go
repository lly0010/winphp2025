package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// pickWebviewDataPath 返回一个仅含 ASCII 字符的 WebView2 用户数据目录.
// Wails 默认会用 %LocalAppData%\<app_name>, 如果 Windows 用户名是中文
// (例如 C:\Users\张三), WebView2 子进程启动可能失败导致主程序无响应.
// 这里按优先级试几个 ASCII 路径, 第一个能用的就用.
func pickWebviewDataPath() string {
	candidates := []string{}
	if pd := os.Getenv("ProgramData"); pd != "" {
		candidates = append(candidates, filepath.Join(pd, "WinPHP", "webview2"))
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		candidates = append(candidates, filepath.Join(la, "WinPHP", "webview2"))
	}
	candidates = append(candidates, `C:\WinPHP-data\webview2`)

	for _, p := range candidates {
		if !isASCII(p) {
			continue
		}
		if err := os.MkdirAll(p, 0o755); err == nil {
			return p
		}
	}
	return ""
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func main() {
	// 初始化目录 (bin, www, logs, tmp, config)
	if err := paths.Init(); err != nil {
		log.Fatalf("paths init: %v", err)
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "WinPHP 2025 - PHP / MySQL / Nginx / PostgreSQL 一键面板",
		Width:             1280,
		Height:            800,
		MinWidth:          1180,
		MinHeight:         720,
		DisableResize:     false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               pickWebviewDataPath(),
			ZoomFactor:                        1.0,
		},
	})

	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
}

