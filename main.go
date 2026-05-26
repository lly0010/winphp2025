package main

import (
	"embed"
	"log"

	"github.com/lly0010/winphp2025/internal/paths"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

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
			WebviewUserDataPath:               "",
			ZoomFactor:                        1.0,
		},
	})

	if err != nil {
		log.Fatalf("wails run: %v", err)
	}
}
